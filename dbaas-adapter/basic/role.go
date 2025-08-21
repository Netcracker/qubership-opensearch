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

const (
	AllIndices                             = "*"
	AttributeResourcePrefix                = "${attr.internal.resource_prefix}*"
	ClusterReadWritePermissions            = "cluster_composite_ops"
	ClusterReadOnlyPermissions             = "cluster_composite_ops_ro"
	ClusterAdminIsmPermissions             = "cluster:admin/opendistro/ism/*"
	ClusterMonitorMainPermission           = "cluster:monitor/main"
	ClusterMonitorHealthPermission         = "cluster:monitor/health"
	ClusterMonitorTaskGetPermission        = "cluster:monitor/task/get"
	ClusterMonitorTaskPermissions          = "cluster:monitor/task*"
	ClusterMonitorStatePermission          = "cluster:monitor/state"
	ClusterScrollClearPermission           = "indices:data/read/scroll/clear"
	ClusterManageIndexTemplatesPermissions = "cluster_manage_index_templates"
	ClusterManageTemplatePermissions       = "indices:admin/template/*"
	ClusterManageIndexTemplatePermissions  = "indices:admin/index_template/*"
	ClusterManageAliasesPermissions        = "indices:admin/aliases/get"
	IndicesIsmManagedIndexPermission       = "indices:admin/opensearch/ism/managedindex"
	IndicesAllActionPermission             = "indices_all"
	IndicesDeletePermission                = "indices:admin/delete"
	IndicesRolloverPermission              = "indices:admin/rollover"
	IndicesMonitorStatsPermission          = "indices:monitor/stats"
	IndicesDMLActionPermission             = "indices:data/*"
	IndicesMappingPutPermission            = "indices:admin/mapping/put"
	IndicesROActionPermission              = "indices:data/read/*"
	IndicesExistPermission                 = "indices:admin/exists"
	IndicesGetPermission                   = "indices:admin/get"
	AdminRoleType                          = "admin"
	DmlRoleType                            = "dml"
	ReadOnlyRoleType                       = "readonly"
	IsmRoleType                            = "ism"
	BackendRolePattern                     = "dbaas_%s"
)

type Role struct {
	ClusterPermissions []string          `json:"cluster_permissions,omitempty"`
	IndexPermissions   []IndexPermission `json:"index_permissions"`
}

type IndexPermission struct {
	IndexPatterns  []string `json:"index_patterns"`
	AllowedActions []string `json:"allowed_actions"`
}

func (bp BaseProvider) GetSupportedRoleTypes() []string {
	return []string{ReadOnlyRoleType, DmlRoleType, AdminRoleType, IsmRoleType}
}

func (bp BaseProvider) DefineRoleType(roleName string) string {
	for _, roleType := range bp.GetSupportedRoleTypes() {
		if strings.Contains(roleName, roleType) {
			return roleType
		}
	}
	return AdminRoleType
}

func (bp BaseProvider) GetBackendRoles(roleType string) []string {
	return []string{fmt.Sprintf(BackendRolePattern, roleType)}
}

func (bp BaseProvider) CreateRoleWithISMPermissions(enhancedSecurityPluginEnabled bool) error {
	clusterPermissions := []string{
		ClusterAdminIsmPermissions,
	}
	indexGlobalPermissions := []string{
		IndicesIsmManagedIndexPermission,
	}
	if !enhancedSecurityPluginEnabled {
		indexGlobalPermissions = append(indexGlobalPermissions,
			IndicesMonitorStatsPermission,
			IndicesRolloverPermission,
			IndicesDeletePermission)
	}
	return bp.createRole(clusterPermissions, []string{}, indexGlobalPermissions, IsmRoleType)
}

func (bp BaseProvider) CreateRoleWithAdminPermissions() error {
	indexPermissions := []string{
		IndicesAllActionPermission,
		strings.ToUpper(IndicesAllActionPermission),
	}
	clusterPermissions := []string{
		ClusterReadWritePermissions,
		strings.ToUpper(ClusterReadWritePermissions),
		ClusterMonitorMainPermission,
		ClusterMonitorHealthPermission,
		ClusterMonitorTaskPermissions,
		ClusterMonitorStatePermission,
		ClusterScrollClearPermission,
		ClusterManageIndexTemplatesPermissions,
		ClusterManageTemplatePermissions,
		ClusterManageIndexTemplatePermissions,
	}
	// IndexGlobalPermissions `indices:admin/resize` permission required for clone index and should be removed after fix https://github.com/opensearch-project/security/issues/429
	indexGlobalPermissions := []string{
		ClusterManageIndexTemplatePermissions,
		ClusterManageAliasesPermissions,
		"indices:admin/resize",
	}
	return bp.createRole(clusterPermissions, indexPermissions, indexGlobalPermissions, AdminRoleType)
}

func (bp BaseProvider) CreateRoleWithDMLPermissions() error {
	indexPermissions := []string{
		IndicesDMLActionPermission,
		strings.ToUpper(IndicesDMLActionPermission),
		IndicesMappingPutPermission,
		strings.ToUpper(IndicesMappingPutPermission),
		IndicesExistPermission,
		strings.ToUpper(IndicesExistPermission),
		IndicesGetPermission,
		strings.ToUpper(IndicesGetPermission),
	}
	clusterPermissions := []string{
		ClusterReadWritePermissions,
		strings.ToUpper(ClusterReadWritePermissions),
		ClusterScrollClearPermission,
		ClusterMonitorTaskGetPermission,
		ClusterMonitorStatePermission,
		ClusterMonitorMainPermission,
	}
	return bp.createRole(clusterPermissions, indexPermissions, []string{}, DmlRoleType)
}

func (bp BaseProvider) CreateRoleWithReadOnlyPermissions() error {
	indexPermissions := []string{
		IndicesROActionPermission,
		IndicesExistPermission,
		IndicesGetPermission,
		strings.ToUpper(IndicesROActionPermission),
		strings.ToUpper(IndicesExistPermission),
		strings.ToUpper(IndicesGetPermission),
	}
	clusterPermissions := []string{
		ClusterReadOnlyPermissions,
		strings.ToUpper(ClusterReadOnlyPermissions),
		ClusterScrollClearPermission,
		ClusterMonitorStatePermission,
		ClusterMonitorMainPermission,
	}
	return bp.createRole(clusterPermissions, indexPermissions, []string{}, ReadOnlyRoleType)
}

func (bp BaseProvider) createRole(clusterPermissions []string, indexPermissions []string,
	globalIndexPermissions []string, roleType string) error {
	name := fmt.Sprintf(common.RoleNamePattern, roleType)
	logger.Debug(fmt.Sprintf("Creating role with name [%s]", name))
	role := Role{
		ClusterPermissions: clusterPermissions,
		IndexPermissions: []IndexPermission{
			{
				IndexPatterns:  []string{AttributeResourcePrefix},
				AllowedActions: indexPermissions,
			},
		},
	}
	if len(globalIndexPermissions) > 0 {
		role.IndexPermissions = append(role.IndexPermissions, IndexPermission{
			IndexPatterns:  []string{AllIndices},
			AllowedActions: globalIndexPermissions,
		})
	}
	body, err := json.Marshal(role)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to marshal body for '%s' role", name))
		return err
	}
	bodyReader := strings.NewReader(string(body))
	header := http.Header{}
	header.Add("Content-type", "application/json")
	createRoleRequest := api.CreateRoleRequest{
		Role:   name,
		Body:   bodyReader,
		Header: header,
	}
	response, err := createRoleRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return fmt.Errorf("error occurred during [%s] role creation: %+v", name, err)
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK || response.StatusCode == http.StatusCreated {
		logger.Info(fmt.Sprintf("'%s' role is successfully created or updated", name))
		return nil
	}

	return fmt.Errorf("role with name [%s] is not created: %+v", name, response.Body)
}

func (bp BaseProvider) GetRole(name string) (*Role, error) {
	logger.Debug(fmt.Sprintf("Getting role with name '%s'", name))
	getRoleRequest := api.GetRoleRequest{
		Role: name,
	}
	response, err := getRoleRequest.Do(context.Background(), bp.opensearch.Client)
	if err != nil {
		return nil, fmt.Errorf("failed to receive role with '%s' name: %+v", name, err)
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		var roles map[string]*Role
		err = common.ProcessBody(response.Body, &roles)
		if err != nil {
			return nil, err
		}
		var role *Role
		for element := range roles {
			role = roles[element]
			break
		}
		return role, nil
	} else if response.StatusCode == http.StatusNotFound {
		logger.Debug(fmt.Sprintf("Role with name '%s' is not found", name))
		return nil, nil
	}
	return nil, fmt.Errorf("during receiving role error occurred: %+v", response.Body)
}
