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
	"testing"
)

func TestUserContentWithResourcePrefix(t *testing.T) {
	username := "admin"
	password := common.GenerateUUID()
	resourcePrefix := common.GetUUID()
	roleType := AdminRoleType
	connectionProperties := common.ConnectionProperties{
		Username:       username,
		Password:       password,
		ResourcePrefix: resourcePrefix,
		Role:           roleType,
	}
	content := bp.getUserContent(connectionProperties)
	expectedAttributes := map[string]string{resourcePrefixAttributeName: resourcePrefix}
	expectedBackendRoles := bp.GetBackendRoles(roleType)
	assert.Equal(t, password, content.Password)
	assert.EqualValues(t, expectedAttributes, content.Attributes)
	assert.EqualValues(t, expectedBackendRoles, content.BackendRoles)
}

func TestUserContentWithoutResourcePrefix(t *testing.T) {
	username := "admin"
	password := common.GenerateUUID()
	roleType := AdminRoleType
	dbName := fmt.Sprintf("%s_test", common.GetUUID())
	connectionProperties := common.ConnectionProperties{
		DbName:   dbName,
		Username: username,
		Password: password,
		Role:     roleType,
	}
	content := bp.getUserContent(connectionProperties)
	expectedAttributes := map[string]string{resourcePrefixAttributeName: dbName}
	expectedBackendRoles := bp.GetBackendRoles(roleType)
	assert.Equal(t, password, content.Password)
	assert.EqualValues(t, expectedAttributes, content.Attributes)
	assert.EqualValues(t, expectedBackendRoles, content.BackendRoles)
}

func TestUserContentWithResourcePrefixAndDbName(t *testing.T) {
	username := "admin"
	password := common.GenerateUUID()
	resourcePrefix := common.GetUUID()
	roleType := AdminRoleType
	dbName := fmt.Sprintf("%s_test", resourcePrefix)
	connectionProperties := common.ConnectionProperties{
		DbName:         dbName,
		Username:       username,
		Password:       password,
		ResourcePrefix: resourcePrefix,
		Role:           roleType,
	}
	content := bp.getUserContent(connectionProperties)
	expectedAttributes := map[string]string{resourcePrefixAttributeName: resourcePrefix}
	expectedBackendRoles := bp.GetBackendRoles(roleType)
	assert.Equal(t, password, content.Password)
	assert.EqualValues(t, expectedAttributes, content.Attributes)
	assert.EqualValues(t, expectedBackendRoles, content.BackendRoles)
}

func TestUserContentWithReadOnlyRole(t *testing.T) {
	username := "admin"
	password := common.GenerateUUID()
	resourcePrefix := common.GetUUID()
	roleType := ReadOnlyRoleType
	connectionProperties := common.ConnectionProperties{
		Username:       username,
		Password:       password,
		ResourcePrefix: resourcePrefix,
		Role:           roleType,
	}
	content := bp.getUserContent(connectionProperties)
	expectedAttributes := map[string]string{resourcePrefixAttributeName: resourcePrefix}
	expectedBackendRoles := bp.GetBackendRoles(roleType)
	assert.Equal(t, password, content.Password)
	assert.EqualValues(t, expectedAttributes, content.Attributes)
	assert.EqualValues(t, expectedBackendRoles, content.BackendRoles)
}

func TestUserContentWithDmlRole(t *testing.T) {
	username := "admin"
	password := common.GenerateUUID()
	resourcePrefix := common.GetUUID()
	roleType := DmlRoleType
	connectionProperties := common.ConnectionProperties{
		Username:       username,
		Password:       password,
		ResourcePrefix: resourcePrefix,
		Role:           roleType,
	}
	content := bp.getUserContent(connectionProperties)
	expectedAttributes := map[string]string{resourcePrefixAttributeName: resourcePrefix}
	expectedBackendRoles := bp.GetBackendRoles(roleType)
	assert.Equal(t, password, content.Password)
	assert.EqualValues(t, expectedAttributes, content.Attributes)
	assert.EqualValues(t, expectedBackendRoles, content.BackendRoles)
}

func TestUserContentWithIsmRole(t *testing.T) {
	username := "admin"
	password := common.GenerateUUID()
	roleType := IsmRoleType
	connectionProperties := common.ConnectionProperties{
		Username: username,
		Password: password,
		Role:     roleType,
	}
	content := bp.getUserContent(connectionProperties)
	expectedBackendRoles := bp.GetBackendRoles(roleType)
	assert.Equal(t, password, content.Password)
	assert.EqualValues(t, expectedBackendRoles, content.BackendRoles)
}

func TestUserContentWithoutRole(t *testing.T) {
	username := "admin"
	password := common.GenerateUUID()
	resourcePrefix := common.GetUUID()
	connectionProperties := common.ConnectionProperties{
		Username:       username,
		Password:       password,
		ResourcePrefix: resourcePrefix,
	}
	content := bp.getUserContent(connectionProperties)
	expectedAttributes := map[string]string{resourcePrefixAttributeName: resourcePrefix}
	expectedBackendRoles := bp.GetBackendRoles(AdminRoleType)
	assert.Equal(t, password, content.Password)
	assert.EqualValues(t, expectedAttributes, content.Attributes)
	assert.EqualValues(t, expectedBackendRoles, content.BackendRoles)
}
