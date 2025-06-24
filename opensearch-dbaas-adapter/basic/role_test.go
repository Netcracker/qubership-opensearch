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
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

const ismRoleWithSecurityPluginType = "ism_with_plugin"

func TestCreateRoleWithAdminPermission(t *testing.T) {
	err := baseProvider.CreateRoleWithAdminPermissions()
	assert.Empty(t, err)
}

func TestGetRoleWithAdminPermission(t *testing.T) {
	name := fmt.Sprintf(common.RoleNamePattern, AdminRoleType)
	clusterPermissions := []string{
		ClusterReadWritePermissions,
		strings.ToUpper(ClusterReadWritePermissions),
		ClusterManageIndexTemplatesPermissions,
		ClusterManageTemplatePermissions,
		ClusterManageIndexTemplatePermissions,
		ClusterMonitorStatePermission,
	}
	indexPermissions := []string{
		IndicesAllActionPermission,
		strings.ToUpper(IndicesAllActionPermission),
	}
	indexGlobalPermissions := []string{
		ClusterManageIndexTemplatePermissions,
		ClusterManageAliasesPermissions,
		"indices:admin/resize",
	}
	role, err := baseProvider.GetRole(name)
	assert.Empty(t, err)
	assert.Equal(t, clusterPermissions, role.ClusterPermissions)
	assert.Len(t, role.IndexPermissions, 2)
	assert.Equal(t, AttributeResourcePrefix, role.IndexPermissions[0].IndexPatterns[0])
	assert.Equal(t, indexPermissions, role.IndexPermissions[0].AllowedActions)
	assert.Equal(t, AllIndices, role.IndexPermissions[1].IndexPatterns[0])
	assert.Equal(t, indexGlobalPermissions, role.IndexPermissions[1].AllowedActions)
}

func TestCreateRoleWithDMLPermission(t *testing.T) {
	err := baseProvider.CreateRoleWithDMLPermissions()
	assert.Empty(t, err)
}

func TestGetRoleWithDMLPermission(t *testing.T) {
	name := fmt.Sprintf(common.RoleNamePattern, DmlRoleType)
	clusterPermissions := []string{
		ClusterReadWritePermissions,
		strings.ToUpper(ClusterReadWritePermissions),
		ClusterMonitorStatePermission,
	}
	indexPermissions := []string{
		IndicesDMLActionPermission,
		strings.ToUpper(IndicesDMLActionPermission),
		IndicesMappingPutPermission,
	}
	role, err := baseProvider.GetRole(name)
	assert.Empty(t, err)
	assert.Equal(t, clusterPermissions, role.ClusterPermissions)
	assert.Len(t, role.IndexPermissions, 1)
	assert.Equal(t, AttributeResourcePrefix, role.IndexPermissions[0].IndexPatterns[0])
	assert.Equal(t, indexPermissions, role.IndexPermissions[0].AllowedActions)
}

func TestCreateRoleWithReadOnlyPermission(t *testing.T) {
	err := baseProvider.CreateRoleWithReadOnlyPermissions()
	assert.Empty(t, err)
}

func TestGetRoleWithReadOnlyPermission(t *testing.T) {
	name := fmt.Sprintf(common.RoleNamePattern, ReadOnlyRoleType)
	clusterPermissions := []string{
		ClusterMonitorStatePermission,
	}
	indexPermissions := []string{
		IndicesROActionPermission,
		strings.ToUpper(IndicesROActionPermission),
	}
	role, err := baseProvider.GetRole(name)
	assert.Empty(t, err)
	assert.Equal(t, clusterPermissions, role.ClusterPermissions)
	assert.Len(t, role.IndexPermissions, 1)
	assert.Equal(t, AttributeResourcePrefix, role.IndexPermissions[0].IndexPatterns[0])
	assert.Equal(t, indexPermissions, role.IndexPermissions[0].AllowedActions)
}

func TestCreateRoleWithISMPermissionAndSecurityPlugin(t *testing.T) {
	err := baseProvider.CreateRoleWithISMPermissions(true)
	assert.Empty(t, err)
}

func TestGetRoleWithISMPermissionAndSecurityPlugin(t *testing.T) {
	name := fmt.Sprintf(common.RoleNamePattern, ismRoleWithSecurityPluginType)
	clusterPermissions := []string{
		ClusterAdminIsmPermissions,
		ClusterMonitorStatePermission,
	}
	indexGlobalPermissions := []string{
		IndicesIsmManagedIndexPermission,
	}
	role, err := baseProvider.GetRole(name)
	assert.Empty(t, err)
	assert.Equal(t, clusterPermissions, role.ClusterPermissions)
	assert.Len(t, role.IndexPermissions, 1)
	assert.Equal(t, AllIndices, role.IndexPermissions[0].IndexPatterns[0])
	assert.Equal(t, indexGlobalPermissions, role.IndexPermissions[0].AllowedActions)
}

func TestCreateRoleWithISMPermissionAndWithoutSecurityPlugin(t *testing.T) {
	err := baseProvider.CreateRoleWithISMPermissions(false)
	assert.Empty(t, err)
}

func TestGetRoleWithISMPermissionAndWithoutSecurityPlugin(t *testing.T) {
	name := fmt.Sprintf(common.RoleNamePattern, IsmRoleType)
	clusterPermissions := []string{
		ClusterAdminIsmPermissions,
		ClusterMonitorStatePermission,
	}

	indexGlobalPermissions := []string{
		IndicesIsmManagedIndexPermission,
		IndicesDeletePermission,
		IndicesRolloverPermission,
		IndicesMonitorStatsPermission,
	}
	role, err := baseProvider.GetRole(name)
	assert.Empty(t, err)
	assert.Equal(t, clusterPermissions, role.ClusterPermissions)
	assert.Len(t, role.IndexPermissions, 1)
	assert.Equal(t, AllIndices, role.IndexPermissions[0].IndexPatterns[0])
	assert.Equal(t, indexGlobalPermissions, role.IndexPermissions[0].AllowedActions)
}
