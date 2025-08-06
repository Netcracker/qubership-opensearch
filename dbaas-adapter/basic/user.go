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
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Netcracker/dbaas-opensearch-adapter/api"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	resourcePrefixAttributeName = "resource_prefix"
	interval                    = 60 * time.Second
	timeout                     = 10 * time.Second
)

type CreatedUser struct {
	ConnectionProperties common.ConnectionProperties `json:"connectionProperties"`
	Name                 string                      `json:"name"`
	Resources            []dao.DbResource            `json:"resources"`
}

type User struct {
	Attributes map[string]string `json:"attributes,omitempty"`
	Hash       string            `json:"hash"`
	Roles      []string          `json:"backend_roles"`
}

type Change struct {
	Operation string      `json:"op"`
	Path      string      `json:"path"`
	Value     interface{} `json:"value"`
}

type Content struct {
	Attributes   map[string]string `json:"attributes,omitempty"`
	BackendRoles []string          `json:"backend_roles,omitempty"`
	Password     string            `json:"password,omitempty"`
}

func (bp BaseProvider) CreateUserHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		username := mux.Vars(r)["name"]
		if username == "" {
			username = fmt.Sprintf("dbaas_%s", common.GenerateUUID())
		}
		logger.InfoContext(ctx, fmt.Sprintf("Request to create user with [%s] name is received", username))

		decoder := json.NewDecoder(r.Body)
		var userCreateRequest dao.UserCreateRequest
		err := decoder.Decode(&userCreateRequest)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request in create database handler", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		response, err := bp.ensureUser(username, userCreateRequest, ctx)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to ensure user", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		responseBody, err := json.Marshal(response)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal response to JSON", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}

		common.ProcessResponseBody(ctx, w, responseBody, http.StatusCreated)
	}
}

func (bp BaseProvider) ensureUser(username string, userCreateRequest dao.UserCreateRequest, ctx context.Context) (*CreatedUser, error) {
	dbName := userCreateRequest.DbName
	roleType := userCreateRequest.Role
	if roleType == "" {
		roleType = AdminRoleType
	}
	username, password, resources, err :=
		bp.createOrUpdateUser(username, userCreateRequest.Password, dbName, roleType, ctx)
	if err != nil {
		return nil, err
	}
	if dbName != "" {
		resources = append(resources, dao.DbResource{Kind: common.MetadataKind, Name: dbName})
		resources = append(resources, dao.DbResource{Kind: common.ResourcePrefixKind, Name: dbName})
	}
	connectionProperties := bp.GetExtendedConnectionProperties("", username, password, "", roleType)
	user, err := bp.GetUser(username)
	if err != nil {
		return nil, err
	}
	if user != nil && user.Attributes[resourcePrefixAttributeName] != "" {
		connectionProperties.ResourcePrefix = user.Attributes[resourcePrefixAttributeName]
	}

	response := &CreatedUser{
		ConnectionProperties: connectionProperties,
		Name:                 dbName,
		Resources:            resources,
	}
	return response, nil
}

// CreateUserByPrefix is used to create user for v2 DBaaS adapter version
func (bp BaseProvider) CreateUserByPrefix(prefix string, password string, dbName string, roleType string, ctx context.Context) (string, string, []dao.DbResource, error) {
	username := fmt.Sprintf("%s_%s", prefix, common.GenerateUUID())
	return bp.createOrUpdateUser(username, password, dbName, roleType, ctx)
}

func (bp BaseProvider) createOrUpdateUser(username string, password string, dbName string, roleType string, ctx context.Context) (string, string, []dao.DbResource, error) {
	bp.mutex.Lock()
	defer bp.mutex.Unlock()
	var resources []dao.DbResource
	if username == "" {
		username = fmt.Sprintf("dbaas_%s", common.GenerateUUID())
	}
	logger.InfoContext(ctx, fmt.Sprintf("Creating user with name '%s' and role '%s'", username, roleType))
	if password == "" {
		var err error
		password, err = bp.passwordGenerator.Generate()
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Cannot generate password for user [%s]", username))
			return username, password, resources, err
		}
	}

	user, err := bp.GetUser(username)
	if err != nil {
		logger.ErrorContext(ctx, fmt.Sprintf("Error occurred during getting user '%s': %v", username, err))
		return username, password, resources, err
	}
	if user == nil {
		err = bp.createUser(username, password, dbName, roleType, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error occurred during creating user '%s': %v", username, err))
			return username, password, resources, fmt.Errorf("during user creation error occurred: %+v", err)
		}
	} else {
		logger.InfoContext(ctx, fmt.Sprintf("Update data for existing [%s] user", username))
		err = bp.PatchUser(username, password, dbName, roleType, ctx)
		if err != nil {
			logger.ErrorContext(ctx, fmt.Sprintf("Error occurred during updating user '%s': %v", username, err))
			return username, password, resources, fmt.Errorf("during user update error occurred: %+v", err)
		}
		logger.InfoContext(ctx, fmt.Sprintf("User [%s] has been updated", username))
	}
	resources = append(resources, dao.DbResource{Kind: common.UserKind, Name: username})
	return username, password, resources, nil
}

func (bp BaseProvider) createUser(username string, password string, prefix string, roleType string, ctx context.Context) error {
	body := Content{Password: password}
	if prefix != "" {
		body.Attributes = map[string]string{resourcePrefixAttributeName: strings.TrimRight(prefix, "*")}
		body.BackendRoles = bp.GetBackendRoles(roleType)
	}
	processedBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	bodyReader := strings.NewReader(string(processedBody))
	header := http.Header{}
	header.Add("Content-type", "application/json")
	userRequest := api.CreateUserRequest{
		Username: username,
		Body:     bodyReader,
		Header:   header,
	}

	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		response, err := userRequest.Do(ctx, bp.opensearch.Client)
		if err != nil {
			logger.ErrorContext(ctx, "Can't process create user request", slog.Any("error", err))
			return false, err
		}
		defer response.Body.Close()
		responseBody, err := io.ReadAll(response.Body)
		logger.DebugContext(ctx, fmt.Sprintf("User response body is %s", responseBody))
		if err != nil {
			return false, nil
		}
		if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated {
			logger.InfoContext(ctx, fmt.Sprintf("User %s is successfully created or updated", username))
			return true, nil
		} else {
			logger.WarnContext(ctx, fmt.Sprintf("Something went wrong during user creation, status code: %d", response.StatusCode))
			return false, nil
		}
	})

	if err != nil {
		return err
	}
	return nil
}

func (bp BaseProvider) PatchUser(username string, password string, prefix string, roleType string, ctx context.Context) error {
	if password == "" && prefix == "" {
		logger.InfoContext(ctx, fmt.Sprintf("There are no parameters to update for [%s] user", username))
		return nil
	}
	var body []Change
	if password != "" {
		body = append(body, Change{Operation: "add", Path: "/password", Value: password})
	}
	if prefix != "" {
		body = append(body, []Change{
			{Operation: "add", Path: "/attributes", Value: map[string]string{resourcePrefixAttributeName: strings.TrimRight(prefix, "*")}},
			{Operation: "add", Path: "/backend_roles", Value: bp.GetBackendRoles(roleType)},
		}...)
	}
	processedBody, err := json.Marshal(body)
	if err != nil {
		return err
	}
	bodyReader := strings.NewReader(string(processedBody))
	header := http.Header{}
	header.Add("Content-type", "application/json")
	patchUserRequest := api.PatchUserRequest{
		Username: username,
		Body:     bodyReader,
		Header:   header,
	}
	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		response, err := patchUserRequest.Do(context.Background(), bp.opensearch.Client)
		if err != nil {
			logger.DebugContext(ctx, "Can't perform patch user request", slog.Any("error", err))
			return false, nil
		}
		responseBody, err := io.ReadAll(response.Body)
		defer response.Body.Close()
		logger.DebugContext(ctx, fmt.Sprintf("User response body is %s", responseBody))
		if err != nil {
			return false, nil
		}
		if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated {
			logger.InfoContext(ctx, fmt.Sprintf("User %s is successfully created or updated", username))
			return true, nil
		} else {
			logger.WarnContext(ctx, fmt.Sprintf("Something went wrong during user update, status code: %d", response.StatusCode))
			return false, nil
		}
	})

	if err != nil {
		return err
	}
	return nil
}

func (bp BaseProvider) patchUsers(changes []Change, ctx context.Context) error {
	if len(changes) == 0 {
		return nil
	}
	processedBody, err := json.Marshal(changes)
	if err != nil {
		return err
	}
	bodyReader := strings.NewReader(string(processedBody))
	header := http.Header{}
	header.Add("Content-type", "application/json")
	patchUsersRequest := api.PatchUsersRequest{
		Body:   bodyReader,
		Header: header,
	}
	response, err := patchUsersRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		logger.InfoContext(ctx, "Batch of users is successfully created or updated")
		return nil
	}
	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("creation of users batch is finished with %d code, response is %s", response.StatusCode,
		string(responseBody))
}

func (bp BaseProvider) GetUser(username string) (*User, error) {
	getUserRequest := api.GetUserRequest{
		Username: username,
	}
	response, err := getUserRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to receive user with '%s' name: %+v", username, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var users map[string]User
		err = common.ProcessBody(response.Body, &users)
		if err != nil {
			return nil, err
		}
		var user User
		for element := range users {
			user = users[element]
			break
		}
		return &user, nil
	} else if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving user error occurred: %+v", response.Body)
}

func (bp BaseProvider) getUsersByPrefix(prefix string) ([]string, error) {
	getUsersRequest := api.GetUsersRequest{}
	response, err := getUsersRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to receive users: %+v", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var users map[string]User
		usersByPrefix := make([]string, 0)
		err = common.ProcessBody(response.Body, &users)
		if err != nil {
			return nil, err
		}
		for element := range users {
			if strings.HasPrefix(element, prefix) {
				usersByPrefix = append(usersByPrefix, element)
			}
		}
		return usersByPrefix, nil
	} else if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving users by prefix %s error occurred: %+v", prefix, response.Body)
}

func (bp BaseProvider) deleteUser(username string, ctx context.Context) error {
	deleteUserRequest := api.DeleteUserRequest{
		Username: username,
	}
	response, err := deleteUserRequest.Do(ctx, bp.opensearch.Client)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	logger.InfoContext(ctx, fmt.Sprintf("User with name [%s] is removed: %+v", username, response.Body))
	return nil
}
