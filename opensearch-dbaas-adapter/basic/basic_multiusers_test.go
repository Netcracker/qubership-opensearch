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
	"github.com/Netcracker/dbaas-opensearch-adapter/cluster"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
)

var (
	opensearchConfiguration = &cluster.Opensearch{
		Host:     "localhost",
		Port:     9200,
		Protocol: common.Http,
		Client:   common.NewClient(),
	}
	bp = BaseProvider{
		opensearch:        opensearchConfiguration,
		mutex:             &sync.Mutex{},
		passwordGenerator: NewPasswordGenerator(),
		ApiVersion:        common.ApiV2,
	}
)

func TestCreateMultiUsersWithoutResourcePrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			CreateOnly: []string{"user"},
		},
	}
	expectedErrorMessage := "'resourcePrefix' must be set to 'true' for v2 version of OpenSearch DBaaS adapter"
	_, err := bp.createDatabase(requestOnCreateDb, ctx)
	assert.NotEmpty(t, err)
	assert.Equal(t, expectedErrorMessage, err.Error())
}

func TestCreateMultiUsersWithResourcePrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: map[string]interface{}{
			"test": "resource_prefix",
		},
		DbName: "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"user"},
		},
	}
	r, err := bp.createDatabase(requestOnCreateDb, ctx)
	assert.Empty(t, err)
	response, ok := r.(DbCreateResponseMultiUser)
	assert.True(t, ok, "casting to DbCreateResponseMultiUser failed")
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Empty(t, response.Name)
	assert.Len(t, response.ConnectionProperties, len(bp.GetSupportedRoleTypes()))
	expectedResources := []dao.DbResource{
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties[0].ResourcePrefix},
		{Kind: common.MetadataKind, Name: response.ConnectionProperties[0].ResourcePrefix},
	}
	for _, connectionProperties := range response.ConnectionProperties {
		assert.NotEmpty(t, connectionProperties.ResourcePrefix)
		assert.Empty(t, connectionProperties.DbName)
		expectedUrl := "http://localhost:9200/"
		assert.Equal(t, expectedUrl, connectionProperties.Url)
		assert.Contains(t, connectionProperties.Username, connectionProperties.ResourcePrefix)
		assert.NotEmpty(t, connectionProperties.Password)
		assert.Subset(t, bp.GetSupportedRoleTypes(), []string{connectionProperties.Role})
		expectedResources = append(expectedResources, dao.DbResource{Kind: common.UserKind, Name: connectionProperties.Username})
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateMultiUsersWithoutCreateOnly(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: map[string]interface{}{
			"test": "without_create_only",
		},
		DbName: "index",
		Settings: Settings{
			ResourcePrefix: true,
		},
	}
	r, err := bp.createDatabase(requestOnCreateDb, ctx)
	assert.Empty(t, err)
	response := r.(DbCreateResponseMultiUser)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Empty(t, response.Name)
	assert.Len(t, response.ConnectionProperties, len(bp.GetSupportedRoleTypes()))
	expectedResources := []dao.DbResource{
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties[0].ResourcePrefix},
		{Kind: common.MetadataKind, Name: response.ConnectionProperties[0].ResourcePrefix},
	}
	for _, connectionProperties := range response.ConnectionProperties {
		assert.NotEmpty(t, connectionProperties.ResourcePrefix)
		assert.Empty(t, connectionProperties.DbName)
		expectedUrl := "http://localhost:9200/"
		assert.Equal(t, expectedUrl, connectionProperties.Url)
		assert.Contains(t, connectionProperties.Username, connectionProperties.ResourcePrefix)
		assert.NotEmpty(t, connectionProperties.Password)
		assert.Subset(t, bp.GetSupportedRoleTypes(), []string{connectionProperties.Role})
		expectedResources = append(expectedResources, dao.DbResource{Kind: common.UserKind, Name: connectionProperties.Username})
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateMultiUsersWithCustomPrefix(t *testing.T) {
	prefix := "dbaas_custom_prefix"
	requestOnCreateDb := DbCreateRequest{
		Metadata: map[string]interface{}{
			"test": "custom_prefix",
		},
		NamePrefix: prefix,
		DbName:     "index",
		Settings: Settings{
			ResourcePrefix: true,
		},
	}
	r, err := bp.createDatabase(requestOnCreateDb, ctx)
	assert.Empty(t, err)
	response := r.(DbCreateResponseMultiUser)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Empty(t, response.Name)
	assert.Len(t, response.ConnectionProperties, len(bp.GetSupportedRoleTypes()))
	expectedResources := []dao.DbResource{
		{Kind: common.ResourcePrefixKind, Name: prefix},
		{Kind: common.MetadataKind, Name: prefix},
	}
	for _, connectionProperties := range response.ConnectionProperties {
		assert.Equal(t, prefix, connectionProperties.ResourcePrefix)
		assert.Empty(t, connectionProperties.DbName)
		expectedUrl := "http://localhost:9200/"
		assert.Equal(t, expectedUrl, connectionProperties.Url)
		assert.Contains(t, connectionProperties.Username, prefix)
		assert.NotEmpty(t, connectionProperties.Password)
		assert.Subset(t, bp.GetSupportedRoleTypes(), []string{connectionProperties.Role})
		expectedResources = append(expectedResources, dao.DbResource{Kind: common.UserKind, Name: connectionProperties.Username})
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestUpdateMultiUserPassword(t *testing.T) {
	username := "7a6f314a-66ee-4999-a841-064b138e163e_5b891494edba4aa5a17568726690c3f2"
	password := "eX5l#RqbdQ"
	userCreateRequest := dao.UserCreateRequest{
		Password: password,
	}
	user, err := bp.ensureUser(username, userCreateRequest, ctx)
	assert.Empty(t, err)
	assert.Empty(t, user.Name)
	assert.Equal(t, "7a6f314a-66ee-4999-a841-064b138e163e", user.ConnectionProperties.ResourcePrefix)
	assert.Empty(t, user.ConnectionProperties.DbName)
	assert.Equal(t, "http://localhost:9200/", user.ConnectionProperties.Url)
	assert.Equal(t, username, user.ConnectionProperties.Username)
	assert.Equal(t, password, user.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: username},
	}
	assert.ElementsMatch(t, user.Resources, expectedResources)
}
