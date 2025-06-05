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
	"github.com/Netcracker/dbaas-opensearch-adapter/api"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"net/http"
	"strings"
)

type RoleMapping struct {
	Users        []string `json:"users,omitempty"`
	Reserved     bool     `json:"reserved,omitempty"`
	BackendRoles []string `json:"backend_roles,omitempty"`
}

func (bp BaseProvider) createRoleMapping(roleName string, roleMapping RoleMapping) error {
	body, err := json.Marshal(roleMapping)
	if err != nil {
		return err
	}
	bodyReader := strings.NewReader(string(body))
	header := http.Header{}
	header.Add("Content-type", "application/json")
	createRolesMappingRequest := api.CreateRolesMappingRequest{
		Role:   roleName,
		Body:   bodyReader,
		Header: header,
	}
	response, err := createRolesMappingRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return fmt.Errorf("failed to create role mapping for '%s' role: %+v", roleName, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		var message map[string]interface{}
		err = common.ProcessBody(response.Body, &message)
		if err != nil {
			return fmt.Errorf("failed to parse response body: %+v", err)
		}
		return fmt.Errorf("failed to create role mapping for [%s] role: [%d] %+v", roleName, response.StatusCode, message)
	}
	logger.Info(fmt.Sprintf("Role mapping for [%s] role is successfully created or updated", roleName))
	return nil
}

func (bp BaseProvider) CreateOrUpdateRoleMapping(roleType string) error {
	roleName := fmt.Sprintf(common.RoleNamePattern, roleType)
	roleMapping, err := bp.GetRoleMapping(roleName)
	if err != nil {
		return err
	}
	if roleMapping == nil {
		roleMapping = &RoleMapping{}
	}
	roleMapping.BackendRoles = bp.GetBackendRolesForMapping(roleType)
	return bp.createRoleMapping(roleName, *roleMapping)
}

func (bp BaseProvider) GetBackendRolesForMapping(roleType string) []string {
	if roleType == AdminRoleType {
		return []string{fmt.Sprintf(BackendRolePattern, AdminRoleType), fmt.Sprintf(BackendRolePattern, IsmRoleType)}
	}
	return []string{fmt.Sprintf(BackendRolePattern, roleType)}
}

func (bp BaseProvider) GetRoleMapping(roleName string) (*RoleMapping, error) {
	logger.Debug(fmt.Sprintf("Getting role mapping for '%s' role", roleName))
	getRoleMappingRequest := api.GetRoleMappingRequest{
		Role: roleName,
	}
	response, err := getRoleMappingRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to receive role mapping for '%s' role: %+v", roleName, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var roleMapping map[string]*RoleMapping
		err = common.ProcessBody(response.Body, &roleMapping)
		if err != nil {
			return nil, err
		}
		return roleMapping[roleName], nil
	} else if response.StatusCode == http.StatusNotFound {
		logger.Debug(fmt.Sprintf("Role mapping with '%s' name is not found", roleName))
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving role mapping error occurred: %+v", response.Body)
}

func (bp BaseProvider) GetRolesMapping() (map[string]RoleMapping, error) {
	logger.Debug("Getting roles mapping")
	var rolesMapping map[string]RoleMapping
	getRolesMappingRequest := api.GetRolesMappingRequest{}
	response, err := getRolesMappingRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return rolesMapping, fmt.Errorf("failed to receive roles mapping: %+v", err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		err = common.ProcessBody(response.Body, &rolesMapping)
		logger.Debug(fmt.Sprintf("The number of roles mapping is %d", len(rolesMapping)))
		return rolesMapping, err
	} else if response.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving roles mapping error occurred: %+v", response.Body)
}
