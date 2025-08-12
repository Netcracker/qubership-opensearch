// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package physical

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Netcracker/dbaas-opensearch-adapter/basic"
	cl "github.com/Netcracker/dbaas-opensearch-adapter/client"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"k8s.io/apimachinery/pkg/util/wait"
)

var logger = common.GetLogger()

type Database struct {
	Id     string            `json:"id"`
	Labels map[string]string `json:"labels"`
}

type ApiVersionInfo struct {
	Major           int   `json:"major,omitempty"`
	Minor           int   `json:"minor,omitempty"`
	SupportedMajors []int `json:"supportedMajors,omitempty"`
}

type RegistrationProvider struct {
	ApiVersion             string
	dbaasAdapter           *common.Component
	dbaasAggregator        *common.Component
	physicalDatabaseId     string
	labelsFileLocation     string
	registrationFixedDelay int
	registrationRetryTime  int
	registrationRetryDelay int
	client                 *http.Client
	Health                 common.ComponentHealth
	status                 dao.Status

	// mutex is used to synchronize concurrent registrations.
	mutex    sync.Mutex
	executor common.BackgroundExecutor

	// baseProvider is used for migration on multi-user approach
	baseProvider *basic.BaseProvider
}

func NewRegistrationProvider(aggregatorAddress string, aggregatorCredentials dao.BasicAuth,
	labelsFileLocation string, client *http.Client, registrationFixedDelay int,
	registrationRetryTime int, registrationRetryDelay int, physicalDatabaseId string,
	adapterAddress string, adapterCredentials dao.BasicAuth, baseProvider *basic.BaseProvider) *RegistrationProvider {
	if client == nil {
		client = cl.ConfigureClient()
	}
	apiVersion := getApiVersion(aggregatorAddress, client)
	baseProvider.ApiVersion = apiVersion
	dbaasAggregator := &common.Component{
		Address:     aggregatorAddress,
		Credentials: aggregatorCredentials,
	}
	dbaasAdapter := &common.Component{
		Address:     adapterAddress,
		Credentials: adapterCredentials,
	}
	return &RegistrationProvider{
		ApiVersion:             apiVersion,
		dbaasAdapter:           dbaasAdapter,
		dbaasAggregator:        dbaasAggregator,
		physicalDatabaseId:     physicalDatabaseId,
		labelsFileLocation:     labelsFileLocation,
		registrationFixedDelay: registrationFixedDelay,
		registrationRetryTime:  registrationRetryTime,
		registrationRetryDelay: registrationRetryDelay,
		client:                 client,
		Health:                 common.ComponentHealth{Status: "UNKNOWN"},
		executor:               common.BackgroundExecutor{},
		status:                 dao.StatusRunning,
		baseProvider:           baseProvider,
	}
}

func getApiVersion(aggregatorAddress string, client *http.Client) string {
	apiVersion := common.GetEnv("API_VERSION", common.ApiV2)
	url := fmt.Sprintf("%s/api-version", aggregatorAddress)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to prepare request to get API version. API %s is enabled by default.", apiVersion), slog.Any("error", err))
		return apiVersion
	}
	response, err := client.Do(request)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to get API version. API %s is enabled by default.", apiVersion), slog.Any("error", err))
		return apiVersion
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var apiVersionInfo ApiVersionInfo
		err = common.ProcessBody(response.Body, &apiVersionInfo)
		if err != nil {
			logger.Error(fmt.Sprintf("Failed to parse api-version response body. API %s is enabled by default.", apiVersion), slog.Any("error", err))
			return apiVersion
		}

		for _, supportedMajor := range apiVersionInfo.SupportedMajors {
			if supportedMajor == 3 {
				logger.Info("Adapter API v2 is enabled.")
				return common.ApiV2
			}
		}
	}
	logger.Info("Adapter API v1 is enabled.")
	return common.ApiV1
}

func (rs *RegistrationProvider) StartRegistration() {
	go rs.registerPeriodically()
}

func (rs *RegistrationProvider) registerPeriodically() {
	for {
		rs.register()
		time.Sleep(time.Duration(rs.registrationFixedDelay) * time.Millisecond)
	}
}

// register performs one physical database registration attempt and sets the corresponding health status
// depending on the result.
func (rs *RegistrationProvider) register() {
	defer rs.mutex.Unlock()
	rs.mutex.Lock()
	rs.doRegistrationRequest()
}

// prepareRequestParameters returns method, URL and body for the registration HTTP request.
// It can cause panic in case of JSON marshalling errors.
func (rs *RegistrationProvider) prepareRequestParameters(ctx context.Context) (string, string, []byte) {
	url := fmt.Sprintf("%s/api/%s/dbaas/opensearch/physical_databases/%s",
		rs.dbaasAggregator.Address, rs.getAggregatorVersion(), rs.physicalDatabaseId)
	registrationRequestBody := dao.PhysicalDatabaseRegistrationRequest{
		AdapterAddress:       rs.dbaasAdapter.Address,
		HttpBasicCredentials: rs.dbaasAdapter.Credentials,
		Labels:               rs.ReadLabelsFile(ctx),
	}
	rs.modifyReqParams(&registrationRequestBody)

	body, err := json.Marshal(registrationRequestBody)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to marshal physical database registration body", slog.Any("error", err))
		panic(err)
	}
	return http.MethodPut, url, body
}

func (rs *RegistrationProvider) modifyReqParams(request *dao.PhysicalDatabaseRegistrationRequest) {
	if rs.ApiVersion == common.ApiV2 {
		request.Metadata = dao.Metadata{
			ApiVersion: dao.ApiVersion(rs.ApiVersion),
			ApiVersions: dao.ApiVersions{Specs: []dao.ApiVersionsSpec{
				{
					SpecRootUrl:     dao.RootUrl,
					Major:           dao.MajorAPIVersion,
					Minor:           dao.MinorAPIVersion,
					SupportedMajors: dao.SupportedMajorsVersions,
				},
			}},
			SupportedRoles: rs.baseProvider.GetSupportedRoleTypes(),
			Features: map[string]bool{
				"multiusers": true,
				"tls":        rs.baseProvider.IsOpenSearchTlsEnabled(),
			},
		}
		request.Status = rs.status
	}
}

// doRegistrationRequest sends an HTTP request to register physical database in DBaaS.
// It causes panic in case of the registration errors.
func (rs *RegistrationProvider) doRegistrationRequest() {
	requestId := common.GenerateUUID()
	ctx := context.WithValue(context.Background(), common.RequestIdKey, requestId)
	defer func() {
		if r := recover(); r != nil {
			message := fmt.Sprintf("%v", r)
			// Uncomment this code when server start doesn't depend on registration and migration processes
			//if strings.Contains(message, "aggregator is available") {
			//	panic(message)
			//}
			logger.InfoContext(ctx, fmt.Sprintf("Recovered from physical database registration panic, set health PROBLEM: %s", message))
			rs.Health = common.ComponentHealth{Status: "PROBLEM"}
		} else {
			logger.InfoContext(ctx, "Successfully registered physical database, set health OK")
			rs.Health = common.ComponentHealth{Status: "OK"}
		}
	}()
	method, url, body := rs.prepareRequestParameters(ctx)
	logger.DebugContext(ctx, fmt.Sprintf("Request parameters are [%s, %s, %s]", method, url, body))
	request, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(body))
	if err != nil {
		logger.ErrorContext(ctx, "Failed to prepare request to register physical database", slog.Any("error", err))
		panic(err)
	}
	request.SetBasicAuth(rs.dbaasAggregator.Credentials.Username, rs.dbaasAggregator.Credentials.Password)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(common.RequestIdKey, requestId)
	response, err := rs.client.Do(request)
	if err != nil || response.StatusCode >= http.StatusBadRequest {
		statusCode, healthError := rs.doHealthRequest()
		if healthError != nil {
			panic(healthError)
		}
		if statusCode >= http.StatusBadRequest {
			panic(fmt.Errorf("aggregator is not available and returns '%d' status code", statusCode))
		}
		panic(fmt.Errorf("the aggregator is available, but the adapter fails to register with '%d' code and error: %v",
			response.StatusCode, err))
	}
	defer response.Body.Close()
	if rs.ApiVersion == common.ApiV2 {
		var physicalDatabaseRegistrationResponse dao.PhysicalDatabaseRegistrationResponse
		if err = common.ProcessBody(response.Body, &physicalDatabaseRegistrationResponse); err != nil {
			panic(err)
		}
		if response.StatusCode != http.StatusOK {
			if err = rs.performMigration(response.StatusCode, physicalDatabaseRegistrationResponse, ctx); err != nil {
				panic(err)
			}
		}
		rs.status = dao.StatusRun
	}
	logger.InfoContext(ctx, "Checked success code for physical database registration")
}

func (rs *RegistrationProvider) doHealthRequest() (int, error) {
	url := fmt.Sprintf("%s/health", rs.dbaasAggregator.Address)
	request, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to prepare request to get aggregator's health: %v", err)
	}
	response, err := rs.client.Do(request)
	if err != nil {
		return http.StatusInternalServerError, fmt.Errorf("failed to get aggregator's health: %v", err)
	}
	defer response.Body.Close()
	return response.StatusCode, nil
}

func (rs *RegistrationProvider) performMigration(statusCode int,
	physicalDatabaseRegistrationResponse dao.PhysicalDatabaseRegistrationResponse,
	ctx context.Context) error {
	logger.InfoContext(ctx, "Starting migration to v3 version of DBaaS aggregator")
	var physicalDatabaseRoleResponse *http.Response
	for statusCode == http.StatusAccepted {
		physicalDatabaseRoleRequest := dao.PhysicalDatabaseRoleRequest{}
		additionalRoles, err := rs.getAdditionalRoles(physicalDatabaseRegistrationResponse, physicalDatabaseRoleResponse)
		if err != nil {
			return err
		}
		for _, additionalRole := range additionalRoles {
			var databaseConnectionProperties []dao.ConnectionProperties
			var databaseResources []dao.DbResource
			receivedProperties := additionalRole.ConnectionProperties
			for _, roleType := range rs.getAbsentRoleTypes(receivedProperties, ctx) {
				connectionProperties, resources, err := rs.createAdditionalResources(additionalRole, roleType, ctx)
				if err != nil {
					logger.ErrorContext(ctx, fmt.Sprintf("Unable to create additional resources because of error: %+v", err))
					physicalDatabaseRoleRequest.Failure = &dao.Failure{
						Id:      additionalRole.Id,
						Message: err.Error(),
					}
					break
				}
				databaseConnectionProperties = append(databaseConnectionProperties, connectionProperties)
				databaseResources = append(databaseResources, resources...)
			}
			if physicalDatabaseRoleRequest.Failure != nil {
				break
			}
			physicalDatabaseRoleRequest.Success = append(physicalDatabaseRoleRequest.Success, dao.Success{
				Id:                   additionalRole.Id,
				ConnectionProperties: databaseConnectionProperties,
				Resources:            databaseResources,
				DbName:               "",
			})
		}

		statusCode, err = rs.performMigrationRequest(ctx, physicalDatabaseRegistrationResponse.Instruction.Id, physicalDatabaseRoleRequest)
		if err != nil {
			return err
		}
	}
	if statusCode == http.StatusInternalServerError {
		return fmt.Errorf("migration is not performed")
	}
	logger.InfoContext(ctx, "Migration to v3 version of DBaaS aggregator is completed successfully")
	return nil
}

func (rs *RegistrationProvider) performMigrationRequest(ctx context.Context, id string, obj dao.PhysicalDatabaseRoleRequest) (int, error) {
	physicalDatabaseRoleResponse, err := rs.doMigrationRequest(id, obj, ctx)
	if err != nil {
		return 0, err
	}
	defer func() {
		err = physicalDatabaseRoleResponse.Body.Close()
		if err != nil {
			logger.Error("failed to close http response body", slog.String("error", err.Error()))
		}
	}()

	statusCode := physicalDatabaseRoleResponse.StatusCode
	return statusCode, nil
}

func (rs *RegistrationProvider) getAdditionalRoles(physicalDatabaseRegistrationResponse dao.PhysicalDatabaseRegistrationResponse,
	response *http.Response) ([]dao.AdditionalRole, error) {
	if response == nil {
		return physicalDatabaseRegistrationResponse.Instruction.AdditionalRoles, nil
	}
	defer response.Body.Close()
	var additionalRoles []dao.AdditionalRole
	err := common.ProcessBody(response.Body, &additionalRoles)
	return additionalRoles, err
}

func (rs *RegistrationProvider) createAdditionalResources(additionalRole dao.AdditionalRole, roleType string, ctx context.Context) (dao.ConnectionProperties, []dao.DbResource, error) {
	logger.InfoContext(ctx, fmt.Sprintf("Creating additional resources for %s role type", roleType))
	var connectionProperties dao.ConnectionProperties
	resourcePrefixProperty := additionalRole.ConnectionProperties[0]["resourcePrefix"]
	if resourcePrefixProperty == nil || resourcePrefixProperty == "" {
		// if `resourcePrefix` property is not specified, use `dbName` as resource_prefix for user
		resourcePrefixProperty = additionalRole.ConnectionProperties[0]["dbName"]
		if resourcePrefixProperty == nil || resourcePrefixProperty == "" {
			return connectionProperties, nil,
				fmt.Errorf("there is no `resourcePrefix` or `dbName` property for additional role with %s ID", additionalRole.Id)
		}
	}
	resourcePrefix := common.ConvertAnyToString(resourcePrefixProperty)
	username, password, resources, err := rs.baseProvider.CreateUserByPrefix(resourcePrefix, "", resourcePrefix, roleType, ctx)
	if err != nil {
		return connectionProperties, nil, err
	}
	extendedConnectionProperties := rs.baseProvider.GetExtendedConnectionProperties("", username, password, resourcePrefix, roleType)
	connectionProperties, err = common.ConvertStructToMap(extendedConnectionProperties)
	if err != nil {
		return connectionProperties, nil, err
	}
	return connectionProperties, resources, nil
}

// getAbsentRoleTypes returns absent role types based on supported role types and data received from DBaaS aggregator
func (rs *RegistrationProvider) getAbsentRoleTypes(receivedProperties []dao.ConnectionProperties, ctx context.Context) []string {
	var absentRoleTypes []string
	for _, roleType := range rs.baseProvider.GetSupportedRoleTypes() {
		absent := true
		for _, properties := range receivedProperties {
			if roleType == properties["role"] {
				absent = false
				break
			}
		}
		if absent {
			absentRoleTypes = append(absentRoleTypes, roleType)
		}
	}
	logger.DebugContext(ctx, fmt.Sprintf("%v roles are to be created for %s prefix", absentRoleTypes, receivedProperties[0]["resourcePrefix"]))
	return absentRoleTypes
}

func (rs *RegistrationProvider) doMigrationRequest(instructionID string, requestBody dao.PhysicalDatabaseRoleRequest, ctx context.Context) (*http.Response, error) {
	url := fmt.Sprintf("%s/api/%s/dbaas/opensearch/physical_databases/%s/instruction/%s/additional-roles",
		rs.dbaasAggregator.Address, rs.getAggregatorVersion(), rs.physicalDatabaseId, instructionID)
	body, err := json.Marshal(requestBody)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to marshal migration body for physical database", slog.Any("error", err))
		return nil, err
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		logger.ErrorContext(ctx, "Failed to prepare request to perform migration for physical database", slog.Any("error", err))
		return nil, err
	}
	request.SetBasicAuth(rs.dbaasAggregator.Credentials.Username, rs.dbaasAggregator.Credentials.Password)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(common.RequestIdKey, common.GetCtxStringValue(ctx, common.RequestIdKey))
	response, err := rs.client.Do(request)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to perform migration for physical database", slog.Any("error", err))
		return nil, err
	}
	return response, nil
}

func (rs *RegistrationProvider) ReadLabelsFile(ctx context.Context) map[string]string {
	var labels map[string]string
	file, err := os.ReadFile(rs.labelsFileLocation)
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("Skipping labels file, cannot read it: %s", rs.labelsFileLocation))
		return labels
	}
	err = json.Unmarshal(file, &labels)
	if err != nil {
		logger.WarnContext(ctx, fmt.Sprintf("Failed to parse labels file %s", rs.labelsFileLocation), slog.Any("error", err))
	}
	logger.DebugContext(ctx, fmt.Sprintf("Read labels: %v", labels))
	return labels
}

func (rs *RegistrationProvider) ForceRegistrationHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, "Received request to force register physical database")
		rs.executor.Submit(rs.RegisterWithRetry)
		w.WriteHeader(http.StatusAccepted)
	}
}

// RegisterWithRetry performs attempts to register physical database in DBaaS during the retryTimeSec.
// If one attempt fails, next attempt is being performed after the retryDelaySec seconds.
// Health status is being updated after each registration attempt.
func (rs *RegistrationProvider) RegisterWithRetry() {
	defer func() {
		if r := recover(); r != nil {
			logger.Info(fmt.Sprintf("Recovered from force physical database registration panic, set health PROBLEM: %v", r))
			rs.Health = common.ComponentHealth{Status: "PROBLEM"}
		}
	}()
	defer rs.mutex.Unlock()

	rs.mutex.Lock()

	interval := time.Duration(rs.registrationRetryDelay) * time.Millisecond
	timeout := time.Duration(rs.registrationRetryTime) * time.Millisecond
	err := wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		rs.doRegistrationRequest()
		return true, nil
	})
	if err != nil {
		logger.Error("Force physical db registration has failed with error", slog.Any("error", err))
		return
	}
	logger.Debug("Force physical db registration finished successfully")
}

func (rs *RegistrationProvider) GetPhysicalDatabaseHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, "Received request to get physical database")
		physicalDatabase := Database{
			Id:     rs.physicalDatabaseId,
			Labels: rs.ReadLabelsFile(ctx),
		}
		responseBody, err := json.Marshal(physicalDatabase)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal physical database response to json", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		common.ProcessResponseBody(ctx, w, responseBody, 0)
	}
}

func (rs *RegistrationProvider) getAggregatorVersion() string {
	dbaasApiVersion := rs.ApiVersion
	dbaasAggregatorVersion := "v1"

	if dbaasApiVersion == common.ApiV2 {
		dbaasAggregatorVersion = "v3"
	}
	return dbaasAggregatorVersion
}
