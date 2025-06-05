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

func TestGetRoleMappingForAdminRole(t *testing.T) {
	name := fmt.Sprintf(common.RoleNamePattern, AdminRoleType)
	roleMapping, err := baseProvider.GetRoleMapping(name)
	assert.Empty(t, err)
	assert.False(t, roleMapping.Reserved)
	assert.Len(t, roleMapping.Users, 10)
}

func TestGetRolesMapping(t *testing.T) {
	rolesMapping, err := baseProvider.GetRolesMapping()
	assert.Empty(t, err)
	assert.Len(t, rolesMapping, 6)
}

func TestUpdateRoleMapping(t *testing.T) {
	err := baseProvider.CreateOrUpdateRoleMapping(AdminRoleType)
	assert.Empty(t, err)
}
