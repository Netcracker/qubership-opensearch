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
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUpdateUserWithDbNameAndPassword(t *testing.T) {
	username := "3439b2b3-b874-44b6-b938-8fa4864740fb"
	userCreateRequest := dao.UserCreateRequest{
		DbName:   "new_test",
		Password: "#NcbMliy9v",
	}
	response, err := baseProvider.ensureUser(username, userCreateRequest, ctx)
	assert.Empty(t, err)
	assert.Equal(t, username, response.ConnectionProperties.ResourcePrefix)
	assert.Equal(t, userCreateRequest.DbName, response.Name)
	assert.Equal(t, "", response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Equal(t, username, response.ConnectionProperties.Username)
	assert.Equal(t, userCreateRequest.Password, response.ConnectionProperties.Password)
	assert.Equal(t, AdminRoleType, response.ConnectionProperties.Role)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: username},
		{Kind: common.MetadataKind, Name: userCreateRequest.DbName},
		{Kind: common.ResourcePrefixKind, Name: userCreateRequest.DbName},
	}
	assert.ElementsMatch(t, expectedResources, response.Resources)
}

func TestCreateUserWithoutUsername(t *testing.T) {
	userCreateRequest := dao.UserCreateRequest{
		DbName:   "new_test",
		Password: "0sjk389ajksl",
	}
	response, err := baseProvider.ensureUser("", userCreateRequest, ctx)
	assert.Empty(t, err)
	assert.Empty(t, response.ConnectionProperties.ResourcePrefix)
	assert.Equal(t, userCreateRequest.DbName, response.Name)
	assert.Equal(t, "", response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, "dbaas_")
	assert.Equal(t, userCreateRequest.Password, response.ConnectionProperties.Password)
	assert.Equal(t, AdminRoleType, response.ConnectionProperties.Role)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.MetadataKind, Name: userCreateRequest.DbName},
		{Kind: common.ResourcePrefixKind, Name: userCreateRequest.DbName},
	}
	assert.ElementsMatch(t, expectedResources, response.Resources)
}

func TestUpdateUserWithDbName(t *testing.T) {
	username := "30fb48f2-ba9a-44d2-8c0a-aad1fcbc0c3d"
	userCreateRequest := dao.UserCreateRequest{
		DbName: "new_test",
	}
	response, err := baseProvider.ensureUser(username, userCreateRequest, ctx)
	assert.Empty(t, err)
	assert.Equal(t, username, response.ConnectionProperties.ResourcePrefix)
	assert.Equal(t, userCreateRequest.DbName, response.Name)
	assert.Equal(t, "", response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Equal(t, username, response.ConnectionProperties.Username)
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	assert.Equal(t, AdminRoleType, response.ConnectionProperties.Role)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: username},
		{Kind: common.MetadataKind, Name: userCreateRequest.DbName},
		{Kind: common.ResourcePrefixKind, Name: userCreateRequest.DbName},
	}
	assert.ElementsMatch(t, expectedResources, response.Resources)
}

func TestUpdateUserWithPassword(t *testing.T) {
	username := "94f5c9a1-f199-4d6d-8d04-0f97527d9842"
	userCreateRequest := dao.UserCreateRequest{
		Password: "PXvm4_VhwJ",
	}
	response, err := baseProvider.ensureUser(username, userCreateRequest, ctx)
	assert.Empty(t, err)
	assert.Equal(t, username, response.ConnectionProperties.ResourcePrefix)
	assert.Empty(t, response.Name)
	assert.Empty(t, response.ConnectionProperties.DbName)
	assert.Equal(t, "http://localhost:9200/", response.ConnectionProperties.Url)
	assert.Equal(t, username, response.ConnectionProperties.Username)
	assert.Equal(t, userCreateRequest.Password, response.ConnectionProperties.Password)
	assert.Equal(t, AdminRoleType, response.ConnectionProperties.Role)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: username},
	}
	assert.ElementsMatch(t, expectedResources, response.Resources)
}

func TestUpdateDmlUserWithPassword(t *testing.T) {
	username := "028cf3fa-4f63-4b86-b038-bce06e5a5e7f_fd4af0e5-38a0-48cc-ad08-a763cbd78013"
	userCreateRequest := dao.UserCreateRequest{
		Password: "_xI3vyFRLO",
		Role:     DmlRoleType,
	}
	response, err := baseProvider.ensureUser(username, userCreateRequest, ctx)
	assert.Empty(t, err)
	assert.Equal(t, "028cf3fa-4f63-4b86-b038-bce06e5a5e7f", response.ConnectionProperties.ResourcePrefix)
	assert.Empty(t, response.Name)
	assert.Empty(t, response.ConnectionProperties.DbName)
	assert.Equal(t, "http://localhost:9200/", response.ConnectionProperties.Url)
	assert.Equal(t, username, response.ConnectionProperties.Username)
	assert.Equal(t, userCreateRequest.Password, response.ConnectionProperties.Password)
	assert.Equal(t, DmlRoleType, response.ConnectionProperties.Role)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: username},
	}
	assert.ElementsMatch(t, expectedResources, response.Resources)
}

func TestGetUser(t *testing.T) {
	username := "a73a026a-da44-4257-aed8-b1ee1bff5b7c"
	response, err := baseProvider.GetUser(username)
	assert.Empty(t, err)
	assert.ElementsMatch(t, []string{username}, response.Roles)
	assert.EqualValues(t, map[string]string{resourcePrefixAttributeName: username}, response.Attributes)
}

func TestDeleteUser(t *testing.T) {
	username := "b5bac0e4-e2d6-45dc-9ae5-77c3b3eb7c8a"
	err := baseProvider.deleteUser(username, ctx)
	assert.Empty(t, err)
}
