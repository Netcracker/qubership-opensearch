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

package basic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"

	"github.com/Netcracker/dbaas-opensearch-adapter/cluster"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	core "github.com/Netcracker/qubership-dbaas-adapter-core/pkg/utils"
	"github.com/gorilla/mux"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
)

const (
	DbaasMetadata        = "dbaas_opensearch_metadata"
	DeletedStatus        = "DELETED"
	DeletionFailedStatus = "DELETE_FAILED"
)

var logger = common.GetLogger()

type BaseProvider struct {
	opensearch        *cluster.Opensearch
	mutex             *sync.Mutex
	passwordGenerator PasswordGenerator
	ApiVersion        string
	recoveryState     string
}

type DbCreateRequest struct {
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
	NamePrefix string                 `json:"namePrefix,omitempty"`
	Password   string                 `json:"password,omitempty"`
	Username   string                 `json:"username,omitempty"`
	DbName     string                 `json:"dbName,omitempty"`
	Settings   Settings               `json:"settings,omitempty"`
}

type Settings struct {
	ResourcePrefix bool        `json:"resourcePrefix,omitempty"`
	CreateOnly     []string    `json:"createOnly,omitempty"`
	IndexSettings  interface{} `json:"indexSettings,omitempty"`
}

type DbCreateResponse struct {
	Name                 string                      `json:"name"`
	ConnectionProperties common.ConnectionProperties `json:"connectionProperties"`
	Resources            []dao.DbResource            `json:"resources"`
}

type DbCreateResponseMultiUser struct {
	Name                 string                        `json:"name"`
	ConnectionProperties []common.ConnectionProperties `json:"connectionProperties"`
	Resources            []dao.DbResource              `json:"resources"`
}

type Metadata struct {
	Found  bool                   `json:"found"`
	Source map[string]interface{} `json:"_source"`
}

type IndexTemplate struct {
	Name          string      `json:"name"`
	IndexTemplate interface{} `json:"index_template"`
}

func NewBaseProvider(opensearch *cluster.Opensearch) *BaseProvider {
	return &BaseProvider{
		opensearch:        opensearch,
		mutex:             &sync.Mutex{},
		passwordGenerator: NewPasswordGenerator(),
		recoveryState:     RecoveryIdleState,
	}
}

func (bp BaseProvider) CreateDatabaseHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, "Request to create new database is received")
		decoder := json.NewDecoder(r.Body)
		var dbCreateRequest DbCreateRequest
		err := decoder.Decode(&dbCreateRequest)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request in create database handler", slog.String("error", err.Error()))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()
		response, err := bp.createDatabase(dbCreateRequest, ctx)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to create database", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed during response serialization in create database handler", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		common.ProcessResponseBody(ctx, w, responseBody, http.StatusCreated)
	}
}

func (bp BaseProvider) ListDatabasesHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, "Request to get indices list is received")
		databases, err := bp.listDatabases()
		if err != nil {
			logger.ErrorContext(ctx, "Failed to get indices list", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		listIndicesBytes, err := json.Marshal(databases)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to serialize indices list", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		common.ProcessResponseBody(ctx, w, listIndicesBytes, http.StatusOK)
	}
}

func (bp BaseProvider) BulkDropResourceHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, "Request to delete OpenSearch resources is received")
		decoder := json.NewDecoder(r.Body)
		var resources []dao.DbResource
		err := decoder.Decode(&resources)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request in delete resources method", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		deletedResources := bp.deleteResources(resources, ctx)
		failedResources := getResourcesWithFailedStatus(deletedResources)
		var resourcesToReturn []dao.DbResource
		if len(failedResources) > 0 {
			resourcesToReturn = failedResources
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			resourcesToReturn = deletedResources
			w.WriteHeader(http.StatusOK)
		}
		bytesResult, err := json.Marshal(resourcesToReturn)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to serialize resources list", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		common.ProcessResponseBody(ctx, w, bytesResult, 0)
	}
}

func (bp BaseProvider) UpdateMetadataHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		indexName := mux.Vars(r)["dbName"]
		logger.InfoContext(ctx, fmt.Sprintf("Request to update metadata for '%s' index is received", indexName))
		var metadata map[string]interface{}
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&metadata)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request in update metadata method", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		_, err = bp.updateMetadata(indexName, metadata, ctx)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to update metadata for index", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (bp BaseProvider) SupportsHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, "Request to get information on supported features is received")
		supports := common.Supports{
			Settings:          true,
			Users:             true,
			DescribeDatabases: false,
		}
		responseBody, err := json.Marshal(supports)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to serialize information about supported features", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		common.ProcessResponseBody(ctx, w, responseBody, 0)
	}
}

func (bp BaseProvider) EnsureAggregationIndex(ctx context.Context) error {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	existsRequest := opensearchapi.IndicesExistsRequest{
		Index: []string{DbaasMetadata},
	}

	childCtx := context.WithValue(ctx, common.RequestIdKey, common.GenerateUUID())
	exist, err := existsRequest.Do(childCtx, bp.opensearch.Client)
	if err != nil {
		logger.ErrorContext(childCtx, fmt.Sprintf("Failed to check if '%s' index exists", DbaasMetadata), slog.String("error", err.Error()))
		return fmt.Errorf("failed to check if '%s' index exists %w", DbaasMetadata, err)
	}

	logger.DebugContext(childCtx, fmt.Sprintf("Check if index exists: %v", exist))
	if exist.StatusCode == 200 {
		logger.DebugContext(childCtx, fmt.Sprintf("'%s' index already exists", DbaasMetadata))
		return nil
	}
	createRequest := opensearchapi.IndicesCreateRequest{
		Index: DbaasMetadata,
	}
	createResponse, err := createRequest.Do(childCtx, bp.opensearch.Client)
	if err != nil {
		exist, err = existsRequest.Do(childCtx, bp.opensearch.Client)
		if err != nil {
			logger.ErrorContext(childCtx, fmt.Sprintf("failed to check if '%s' index exists", DbaasMetadata), slog.Any("error", err))
			return fmt.Errorf("failed to check if '%s' index exists %w", DbaasMetadata, err)
		}
		logger.DebugContext(childCtx, fmt.Sprintf("Check if index exists: %v", exist))
		if exist.StatusCode == 200 {
			logger.DebugContext(childCtx, fmt.Sprintf("'%s' index already exists", DbaasMetadata))
			return nil
		}
		logger.ErrorContext(childCtx, fmt.Sprintf("failed to create '%s' index", DbaasMetadata), slog.Any("error", err))
		return fmt.Errorf("failed to create '%s' index %w", DbaasMetadata, err)
	}
	defer createResponse.Body.Close()

	if createResponse.StatusCode != http.StatusCreated && createResponse.StatusCode != http.StatusOK {
		var body []byte
		body, err = io.ReadAll(createResponse.Body)
		if err != nil {
			logger.ErrorContext(childCtx, "failed to read from http response body", slog.String("error", err.Error()))
			return err
		}
		logger.ErrorContext(childCtx, fmt.Sprintf("%s index cannot be created because of error: [%d] %s", DbaasMetadata,
			createResponse.StatusCode, string(body)))
		return fmt.Errorf("%s index cannot be created because of error: [%d]", DbaasMetadata,
			createResponse.StatusCode)
	}
	logger.DebugContext(childCtx, fmt.Sprintf("'%s' index is created", DbaasMetadata))
	return nil
}

func (bp BaseProvider) createDatabase(requestOnCreateDb DbCreateRequest, ctx context.Context) (interface{}, error) {
	var resources []dao.DbResource
	var connections []common.ConnectionProperties
	var prefix string
	var namespace string
	var microserviceName string
	logger.InfoContext(ctx, fmt.Sprintf("Creating new database for requests, dbName: '%s', username: '%s', metadata: '%+v', settings: '%+v'",
		requestOnCreateDb.DbName, requestOnCreateDb.Username, requestOnCreateDb.Metadata, requestOnCreateDb.Settings))
	if classifier, ok := requestOnCreateDb.Metadata["classifier"]; ok {
		var classifierMap map[string]interface{}
		classifierMap, ok = classifier.(map[string]interface{})
		if ok {
			var requestNamespace interface{}
			if requestNamespace, ok = classifierMap["namespace"]; ok {
				namespace = common.ConvertAnyToString(requestNamespace)
			}
		}

	}
	if requestMicroserviceName, ok := requestOnCreateDb.Metadata["microserviceName"]; ok {
		microserviceName = common.ConvertAnyToString(requestMicroserviceName)
	}

	if requestOnCreateDb.Settings.ResourcePrefix {
		err := checkForbiddenSymbolPrefix(requestOnCreateDb.NamePrefix)
		if err != nil {
			return nil, err
		}
		if len(requestOnCreateDb.NamePrefix) == 0 {
			if len(namespace) == 0 || len(microserviceName) == 0 {
				prefix = common.GetUUID()
			} else {
				prefix, err = core.PrepareDatabaseName(namespace, microserviceName, 64)
				if err != nil {
					return nil, err
				}
				err = checkForbiddenSymbolPrefix(prefix)
				if err != nil {
					logger.ErrorContext(ctx, fmt.Sprintf("Database prefix contains forbidden symbols: %s", prefix), slog.Any("error", err))
					return nil, err
				}
			}
		} else {
			prefix = requestOnCreateDb.NamePrefix
		}
		resources = append(resources, dao.DbResource{Kind: common.ResourcePrefixKind, Name: prefix})
	} else {
		if bp.ApiVersion == common.ApiV2 {
			return nil, fmt.Errorf("'resourcePrefix' must be set to 'true' for v2 version of OpenSearch DBaaS adapter")
		}
		prefix = requestOnCreateDb.NamePrefix
		if prefix == "" {
			prefix = "dbaas"
		}
	}

	if ok, err := common.CheckPrefixUniqueness(prefix, ctx, bp.opensearch.Client); !ok {
		if err != nil {
			return nil, err
		}
	}

	resourcesToCreate := requestOnCreateDb.Settings.CreateOnly
	if len(resourcesToCreate) == 0 {
		resourcesToCreate = []string{common.UserKind, common.IndexKind}
		if bp.ApiVersion == common.ApiV2 {
			resourcesToCreate = []string{common.UserKind}
		}
	}

	logger.InfoContext(ctx, fmt.Sprintf("Creating the following resource for database '%t': [%v]",
		requestOnCreateDb.Settings.ResourcePrefix, resourcesToCreate))

	var indexName string
	var username string
	var password string
	var err error
	for _, resource := range resourcesToCreate {
		if resource == common.IndexKind {
			indexName, err = bp.createIndex(requestOnCreateDb, prefix, ctx)
			if err != nil {
				return nil, err
			}
			resources = append(resources, dao.DbResource{Kind: common.IndexKind, Name: indexName})
		}
		if resource == common.UserKind {
			var dbName string
			username = requestOnCreateDb.Username
			if requestOnCreateDb.Settings.ResourcePrefix {
				username = prefix
				dbName = fmt.Sprintf("%s*", prefix)
			} else {
				dbName = buildIndexName(requestOnCreateDb.DbName, prefix)
			}
			var securityResources []dao.DbResource
			if bp.ApiVersion == common.ApiV1 {
				username, password, securityResources, err =
					bp.createOrUpdateUser(username, requestOnCreateDb.Password, dbName, AdminRoleType, ctx)
				if err != nil {
					return nil, err
				}
				resources = append(resources, securityResources...)
			}
			// Possibly need to move additionalRoles and response logic into separate methods for v2
			// and check apiVersion once to improve readability
			if bp.ApiVersion == common.ApiV2 {
				for _, roleType := range bp.GetSupportedRoleTypes() {
					additionalUsername := prefix
					additionalPassword := ""
					additionalUsername, additionalPassword, securityResources, err =
						bp.CreateUserByPrefix(additionalUsername, additionalPassword, dbName, roleType, ctx)
					if err != nil {
						return nil, err
					}
					connectionProperties := bp.GetExtendedConnectionProperties(indexName, additionalUsername,
						additionalPassword, prefix, roleType)
					connections = append(connections, connectionProperties)
					resources = append(resources, securityResources...)
				}
			}
		}
	}

	metadataID := prefix
	if indexName != "" {
		metadataID = indexName
	}
	_, err = bp.CreateMetadata(metadataID, requestOnCreateDb.Metadata, ctx)
	if err != nil {
		if indexName != "" {
			err = bp.deleteDatabase(indexName, ctx)
			if err != nil {
				return nil, err
			}
		}
		return nil, err
	}
	resources = append(resources, dao.DbResource{Kind: common.MetadataKind, Name: metadataID})

	var result interface{}
	if bp.ApiVersion == common.ApiV1 {
		connection := bp.getConnectionProperties(indexName, username, password)
		response := DbCreateResponse{Name: indexName, ConnectionProperties: connection, Resources: resources}
		if requestOnCreateDb.Settings.ResourcePrefix {
			response.ConnectionProperties.ResourcePrefix = prefix
		}
		result = response
	} else if bp.ApiVersion == common.ApiV2 {
		response := DbCreateResponseMultiUser{Name: indexName, ConnectionProperties: connections, Resources: resources}
		result = response
	}

	return result, nil
}

func checkForbiddenSymbolPrefix(namePrefix string) error {
	if strings.HasPrefix(namePrefix, ".") || strings.Contains(namePrefix, "*") {
		return errors.New("prefix contains forbidden symbols")
	}
	return nil
}

func (bp BaseProvider) createIndex(requestOnCreateDb DbCreateRequest, prefix string, ctx context.Context) (string, error) {
	indexName := buildIndexName(requestOnCreateDb.DbName, prefix)
	body := strings.NewReader(getIndexSettings(&requestOnCreateDb, ctx))
	indexRequest := opensearchapi.IndicesCreateRequest{
		Index: indexName,
		Body:  body,
	}
	logger.InfoContext(ctx, fmt.Sprintf("Creating index with name '%s'", indexName))
	indexResponse, err := indexRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("Error occurred during creating '%s' index", indexName), slog.Any("error", err))
		return indexName, err
	}
	logger.InfoContext(ctx, fmt.Sprintf("Index with name '%s' is created: %s", indexName, indexResponse.Body))
	return indexName, nil
}

func (bp BaseProvider) getDatabase(name string) (interface{}, error) {
	getRequest := opensearchapi.IndicesGetRequest{
		Index: []string{name},
	}
	response, err := getRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to receive index with '%s' name: %+v", name, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var indices map[string]interface{}
		err = common.ProcessBody(response.Body, &indices)
		if err != nil {
			return nil, err
		}
		var index interface{}
		for element := range indices {
			index = indices[element]
			break
		}
		return index, nil
	} else if response.StatusCode == http.StatusNotFound {
		// Database is not found without errors
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving index error occurred: %+v", response.Body)
}

func (bp BaseProvider) listDatabases() ([]string, error) {
	indicesRequest := opensearchapi.CatIndicesRequest{
		H: []string{"index"},
	}
	response, err := indicesRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, fmt.Errorf("error occurred during retrieving indices list: %+v", err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("error occurred during retrieving indices list: %+v", err)
	}
	// Indices filtration: inner (".") indices should not be returned by the request
	var indices []string
	for _, index := range strings.Split(string(body), "\n") {
		if index != "" && !strings.HasPrefix(index, ".") {
			indices = append(indices, index)
		}
	}
	return indices, nil
}

func (bp BaseProvider) deleteDatabase(name string, ctx context.Context) error {
	indicesDeleteRequest := opensearchapi.IndicesDeleteRequest{
		Index: []string{name},
	}
	response, err := indicesDeleteRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	logger.DebugContext(ctx, fmt.Sprintf("Index with name [%s] is removed: %+v", name, response.Body))
	return nil
}

func (bp BaseProvider) GetMetadata(indexName string, ctx context.Context) (map[string]interface{}, error) {
	logger.InfoContext(ctx, fmt.Sprintf("Get metadata for '%s' index", indexName))
	getRequest := opensearchapi.GetRequest{
		Index:      DbaasMetadata,
		DocumentID: indexName,
	}
	var response Metadata
	err := common.DoRequest(getRequest, bp.opensearch.Client, &response, ctx)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("Error occurred during get metadata for '%s' index", indexName), slog.Any("error", err))
		return nil, err
	}
	if !response.Found {
		logger.InfoContext(ctx, fmt.Sprintf("Metadata is not found for '%s' index", indexName))
		return nil, nil
	}
	return response.Source, nil
}

func (bp BaseProvider) CreateMetadata(identifier string, metadata map[string]interface{}, ctx context.Context) (string, error) {
	logger.InfoContext(ctx, fmt.Sprintf("Insert metadata to '%s' index by '%s' identifier", DbaasMetadata, identifier))
	if metadata == nil {
		return "", nil
	}
	metadataJson, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	body := strings.NewReader(string(metadataJson))
	indexRequest := opensearchapi.IndexRequest{
		Index:      DbaasMetadata,
		DocumentID: identifier,
		Body:       body,
	}
	response, err := indexRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return "", fmt.Errorf("error occurred during insert metadata for '%s' ID to '%s' index : %+v", identifier, DbaasMetadata, err)
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	logger.InfoContext(ctx, fmt.Sprintf("%d: %s", response.StatusCode, string(responseBody)))
	return string(responseBody), nil
}

func (bp BaseProvider) updateMetadata(indexName string, metadata map[string]interface{}, ctx context.Context) (string, error) {
	if !bp.ensureMetadata(indexName, metadata, ctx) {
		metadataBytes, err := json.Marshal(metadata)
		if err != nil {
			return "", err
		}
		bodyReader := strings.NewReader(string(metadataBytes))
		updateRequest := opensearchapi.UpdateRequest{
			Index:      DbaasMetadata,
			DocumentID: indexName,
			Body:       bodyReader,
		}
		response, err := updateRequest.Do(context.Background(), bp.opensearch.Client)
		if err != nil {
			logger.ErrorContext(ctx, "Error occurred during update metadata", slog.Any("error", err))
			return "", err
		}
		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}
		return string(responseBody), nil
	}
	return "", nil
}

func (bp BaseProvider) ensureMetadata(indexName string, metadata map[string]interface{}, ctx context.Context) (ret bool) {
	ret = false
	source, err := bp.GetMetadata(indexName, ctx)
	if err != nil || source == nil {
		_, err = bp.CreateMetadata(indexName, metadata, ctx)
		if err != nil {
			return
		}
		ret = true
	}
	return
}

func (bp BaseProvider) deleteMetadata(indexName string, ctx context.Context) error {
	deleteMetadataRequest := opensearchapi.DeleteRequest{
		Index:      DbaasMetadata,
		DocumentID: indexName,
	}
	response, err := deleteMetadataRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	logger.DebugContext(ctx, fmt.Sprintf("Metadata for index with name [%s] is removed: %+v", indexName, response.Body))
	return nil
}

func (bp BaseProvider) getTemplate(name string) (interface{}, error) {
	getTemplateRequest := opensearchapi.IndicesGetTemplateRequest{
		Name: []string{name},
	}
	response, err := getTemplateRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var templates map[string]interface{}
		err = common.ProcessBody(response.Body, &templates)
		if err != nil {
			return nil, err
		}
		var template interface{}
		for element := range templates {
			template = templates[element]
			break
		}
		return template, nil
	} else if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving template error occurred: %+v", response.Body)
}

func (bp BaseProvider) getIndexTemplate(name string) (*IndexTemplate, error) {
	getIndexTemplateRequest := opensearchapi.IndicesGetIndexTemplateRequest{
		Name: []string{name},
	}
	response, err := getIndexTemplateRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var templates map[string][]IndexTemplate
		err = common.ProcessBody(response.Body, &templates)
		if err != nil {
			return nil, err
		}
		template := templates["index_templates"][0]
		return &template, nil
	} else if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("during receivingindex  template error occurred: %+v", response.Body)
}

func (bp BaseProvider) deleteTemplate(template string, ctx context.Context) error {
	deleteTemplateRequest := opensearchapi.IndicesDeleteTemplateRequest{
		Name: template,
	}
	response, err := deleteTemplateRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	logger.DebugContext(ctx, fmt.Sprintf("Template with name [%s] is removed: %+v", template, response.Body))
	return nil
}

func (bp BaseProvider) deleteIndexTemplate(template string, ctx context.Context) error {
	deleteIndexTemplateRequest := opensearchapi.IndicesDeleteIndexTemplateRequest{
		Name: template,
	}
	response, err := deleteIndexTemplateRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	logger.DebugContext(ctx, fmt.Sprintf("Index Template with name [%s] is removed: %+v", template, response.Body))
	return nil
}

func (bp BaseProvider) getAlias(name string) (interface{}, error) {
	getAliasRequest := opensearchapi.IndicesGetAliasRequest{
		Name: []string{name},
	}
	response, err := getAliasRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var aliases map[string]map[string]interface{}
		err = common.ProcessBody(response.Body, &aliases)
		if err != nil {
			return nil, err
		}
		var alias map[string]interface{}
		for element := range aliases {
			alias = aliases[element]
			break
		}
		return &alias, nil
	} else if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving alias error occurred: %+v", response.Body)
}

func (bp BaseProvider) deleteAlias(alias string, ctx context.Context) error {
	aliasDeleteTemplate := opensearchapi.IndicesDeleteAliasRequest{
		Index: []string{alias},
		Name:  []string{alias},
	}
	response, err := aliasDeleteTemplate.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	logger.DebugContext(ctx, fmt.Sprintf("Alias with name [%s] is removed: %+v", alias, response.Body))
	return nil
}

func (bp BaseProvider) getConnectionProperties(dbName string, username string, password string) common.ConnectionProperties {
	url := fmt.Sprintf("%s://%s:%d/%s", bp.opensearch.Protocol, bp.opensearch.Host, bp.opensearch.Port, dbName)
	return common.ConnectionProperties{
		DbName:   dbName,
		Host:     bp.opensearch.Host,
		Port:     bp.opensearch.Port,
		Url:      url,
		Username: username,
		Password: password,
	}
}

func (bp BaseProvider) IsOpenSearchTlsEnabled() bool {
	return bp.ApiVersion == common.ApiV2 && bp.opensearch.Protocol == common.Https
}

func (bp BaseProvider) GetExtendedConnectionProperties(dbName string, username string, password string, prefix string,
	roleType string) common.ConnectionProperties {
	connectionProperties := bp.getConnectionProperties(dbName, username, password)
	if prefix != "" {
		connectionProperties.ResourcePrefix = prefix
	}
	if roleType != "" {
		connectionProperties.Role = roleType
	}
	return connectionProperties
}

func (bp BaseProvider) deleteResources(resources []dao.DbResource, ctx context.Context) []dao.DbResource {
	var deletedResources []dao.DbResource

	resources = append(resources, bp.processResourcePrefixKind(resources, ctx)...)

	users := bp.deleteResourcesByKind(resources, common.UserKind)
	deletedResources = append(deletedResources, users...)

	databases := bp.deleteResourcesByKind(resources, common.IndexKind)
	deletedResources = append(deletedResources, databases...)

	metadata := bp.deleteResourcesByKind(resources, common.MetadataKind)
	deletedResources = append(deletedResources, metadata...)

	templates := bp.deleteResourcesByKind(resources, common.TemplateKind)
	deletedResources = append(deletedResources, templates...)

	indexTemplates := bp.deleteResourcesByKind(resources, common.IndexTemplateKind)
	deletedResources = append(deletedResources, indexTemplates...)

	aliases := bp.deleteResourcesByKind(resources, common.AliasKind)
	deletedResources = append(deletedResources, aliases...)

	return deletedResources
}

func (bp BaseProvider) processResourcePrefixKind(resources []dao.DbResource, ctx context.Context) []dao.DbResource {
	var additionalResources []dao.DbResource
	for _, resource := range resources {
		if resource.Kind == common.ResourcePrefixKind {
			namePattern := fmt.Sprintf("%s*", resource.Name)
			if bp.ApiVersion == common.ApiV1 {
				additionalResources = append(additionalResources, []dao.DbResource{
					{Kind: common.UserKind, Name: resource.Name},
					{Kind: common.IndexKind, Name: namePattern},
					{Kind: common.MetadataKind, Name: resource.Name},
					{Kind: common.TemplateKind, Name: namePattern},
					{Kind: common.IndexTemplateKind, Name: namePattern},
					{Kind: common.AliasKind, Name: namePattern},
				}...)
			} else if bp.ApiVersion == common.ApiV2 {
				additionalResources = append(additionalResources, []dao.DbResource{
					{Kind: common.IndexKind, Name: namePattern},
					{Kind: common.MetadataKind, Name: resource.Name},
					{Kind: common.TemplateKind, Name: namePattern},
					{Kind: common.IndexTemplateKind, Name: namePattern},
					{Kind: common.AliasKind, Name: namePattern},
				}...)
				users, err := bp.getUsersByPrefix(resource.Name)
				if err != nil {
					logger.ErrorContext(ctx, fmt.Sprintf("Failed to receive users with prefix %s ", resource.Name), slog.Any("error", err))
				} else {
					for _, user := range users {
						additionalResources = append(additionalResources,
							dao.DbResource{Kind: common.UserKind, Name: user})
					}
				}
			}
		}
	}
	return additionalResources
}

func (bp BaseProvider) deleteResourcesByKind(resources []dao.DbResource, kind string) []dao.DbResource {
	var result []dao.DbResource
	for _, resource := range resources {
		if resource.Kind == kind {
			deletedResource := bp.deleteResource(resource, context.Background())
			result = append(result, *deletedResource)
		}
	}
	return result
}

func (bp BaseProvider) deleteResource(resource dao.DbResource, ctx context.Context) *dao.DbResource {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()

	if resource.Kind == common.IndexKind {
		database, err := bp.getDatabase(resource.Name)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to receive '%s' index information", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
		if database == nil {
			logger.InfoContext(ctx, fmt.Sprintf("'%s' index does not exist, skip deletion", resource.Name))
			return getResourceDeletionSuccessStatus(resource)
		}
		err = bp.deleteDatabase(resource.Name, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to delete '%s' index", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
	} else if resource.Kind == common.MetadataKind {
		metadata, err := bp.GetMetadata(resource.Name, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to receive metadata for '%s' index", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
		if metadata == nil {
			logger.InfoContext(ctx, fmt.Sprintf("Metadata for '%s' index does not exist, skip deletion", resource.Name))
			return getResourceDeletionSuccessStatus(resource)
		}
		err = bp.deleteMetadata(resource.Name, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to delete metadata for '%s' index", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
	} else if resource.Kind == common.UserKind {
		user, err := bp.GetUser(resource.Name)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to receive '%s' user information", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
		if user == nil {
			logger.InfoContext(ctx, fmt.Sprintf("'%s' user does not exist, skip deletion", resource.Name))
			return getResourceDeletionSuccessStatus(resource)
		}
		err = bp.deleteUser(resource.Name, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to delete '%s' user", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
	} else if resource.Kind == common.TemplateKind {
		template, err := bp.getTemplate(resource.Name)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to receive '%s' template information", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
		if template == nil {
			logger.InfoContext(ctx, fmt.Sprintf("'%s' template does not exist, skip deletion", resource.Name))
			return getResourceDeletionSuccessStatus(resource)
		}
		err = bp.deleteTemplate(resource.Name, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to delete '%s' template", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
	} else if resource.Kind == common.IndexTemplateKind {
		template, err := bp.getIndexTemplate(resource.Name)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to receive '%s' index template information", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
		if template == nil {
			logger.InfoContext(ctx, fmt.Sprintf("'%s' index template does not exist, skip deletion", resource.Name))
			return getResourceDeletionSuccessStatus(resource)
		}
		err = bp.deleteIndexTemplate(resource.Name, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to delete '%s' index template", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
	} else if resource.Kind == common.AliasKind {
		alias, err := bp.getAlias(resource.Name)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to receive '%s' alias information", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
		if alias == nil {
			logger.InfoContext(ctx, fmt.Sprintf("'%s' alias does not exist, skip deletion", resource.Name))
			return getResourceDeletionSuccessStatus(resource)
		}
		err = bp.deleteAlias(resource.Name, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Failed to delete '%s' alias", resource.Name), slog.Any("error", err))
			return getResourceDeletionFailedStatus(resource, err)
		}
	}
	return getResourceDeletionSuccessStatus(resource)
}

func getResourceDeletionSuccessStatus(resource dao.DbResource) *dao.DbResource {
	return &dao.DbResource{
		Kind:   resource.Kind,
		Name:   resource.Name,
		Status: DeletedStatus,
	}
}

func getResourceDeletionFailedStatus(resource dao.DbResource, err error) *dao.DbResource {
	return &dao.DbResource{
		Kind:         resource.Kind,
		Name:         resource.Name,
		Status:       DeletionFailedStatus,
		ErrorMessage: err.Error(),
	}
}

func getResourcesWithFailedStatus(resources []dao.DbResource) []dao.DbResource {
	var result []dao.DbResource
	for _, resource := range resources {
		if resource.Status == DeletionFailedStatus {
			result = append(result, resource)
		}
	}
	return result
}

func buildIndexName(dbName string, prefix string) string {
	var indexName string
	if dbName == "" {
		dbName = common.GetUUID()
	}
	indexName = fmt.Sprintf("%s_%s", prefix, dbName)
	return indexName
}

func getIndexSettings(createDbRequest *DbCreateRequest, ctx context.Context) string {
	if createDbRequest.Settings.IndexSettings != nil {
		createIndexSettings, err := json.Marshal(createDbRequest.Settings.IndexSettings)
		if err != nil {
			logger.ErrorContext(ctx, "Failed during serializable index settings", slog.Any("error", err))
			return ""
		}
		return string(createIndexSettings[:])
	}
	return ""
}
