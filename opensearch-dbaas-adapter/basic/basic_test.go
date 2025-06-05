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
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/cluster"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
	"github.com/stretchr/testify/assert"
	"os"
	"sync"
	"testing"
)

var baseProvider BaseProvider
var ctx context.Context

func TestMain(m *testing.M) {
	setup()
	code := m.Run()
	shutdown()
	os.Exit(code)
}

func setup() {
	opensearchClient := common.NewClient()
	opensearch := &cluster.Opensearch{
		Host:     "localhost",
		Port:     9200,
		Protocol: common.Http,
		Client:   opensearchClient,
	}
	baseProvider = BaseProvider{
		opensearch:        opensearch,
		mutex:             &sync.Mutex{},
		passwordGenerator: NewPasswordGenerator(),
		ApiVersion:        common.ApiV1,
	}
	ctx = context.WithValue(context.Background(), common.RequestIdKey, common.GenerateUUID())
}

func shutdown() {
	baseProvider.opensearch.Client = nil
}

func TestGetIndex(t *testing.T) {
	indexName := "dbaas_metadata"
	index, err := baseProvider.getDatabase(indexName)
	assert.Empty(t, err)
	assert.NotEmpty(t, index)
}

func TestListIndices(t *testing.T) {
	indices, err := baseProvider.listDatabases()
	assert.Empty(t, err)
	expectedIndices := []string{"dbaas_metadata", "dbaas_opensearch_metadata", "testmine", "test-new", "testme"}
	assert.ElementsMatch(t, indices, expectedIndices)
}

func TestDeleteIndex(t *testing.T) {
	indexName := "testing"
	err := baseProvider.deleteDatabase(indexName, context.Background())
	assert.Empty(t, err)
}

func TestCreateMetadata(t *testing.T) {
	indexName := "sjkao-new-index"
	metadata := map[string]interface{}{"doc": map[string]string{"key": "value"}}
	response, err := baseProvider.CreateMetadata(indexName, metadata, ctx)
	assert.Empty(t, err)
	assert.Contains(t, response, indexName)
	assert.Contains(t, response, `"result":"created"`)
}

func TestGetMetadata(t *testing.T) {
	source, err := baseProvider.GetMetadata("qbjwk-dbaas-index", ctx)
	assert.Empty(t, err)
	assert.Equal(t, "check", source["text"])
}

func TestUpdateMetadata(t *testing.T) {
	indexName := "jsleu-new-index"
	metadata := map[string]interface{}{"doc": map[string]string{"key": "value"}}
	response, err := baseProvider.updateMetadata(indexName, metadata, ctx)
	assert.Empty(t, err)
	assert.Contains(t, response, indexName)
	assert.Contains(t, response, `"result":"updated"`)
}

func TestDeleteMetadata(t *testing.T) {
	err := baseProvider.deleteMetadata("pbemw-dbaas-index", context.Background())
	assert.Empty(t, err)
}

func TestCreateIndexWithForbiddenDotPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata:   nil,
		NamePrefix: ".test_Prefix",
		DbName:     "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"index"},
		},
	}
	_, err := baseProvider.createDatabase(requestOnCreateDb, ctx)
	assert.Error(t, err)
	assert.Equal(t, "prefix contains forbidden symbols", err.Error())
}

func TestCreateIndexWithForbiddenAsteriskPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata:   nil,
		NamePrefix: "test_Prefix*",
		DbName:     "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"index"},
		},
	}
	_, err := baseProvider.createDatabase(requestOnCreateDb, ctx)
	assert.Error(t, err)
	assert.Equal(t, "prefix contains forbidden symbols", err.Error())
}

func TestCreateIndexWithCustomPrefix(t *testing.T) {
	var namePrefix = "some_name"
	requestOnCreateDb := DbCreateRequest{
		Metadata:   nil,
		NamePrefix: namePrefix,
		DbName:     "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"index"},
		},
	}
	r, err := baseProvider.createDatabase(requestOnCreateDb, ctx)
	assert.NoError(t, err, "failed to create database")
	response, ok := r.(DbCreateResponse)
	assert.True(t, ok, "failed to cast type DbCreateResponse")
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Equal(t, namePrefix, response.ConnectionProperties.ResourcePrefix)
	expectedIndexName := fmt.Sprintf("%s_%s", response.ConnectionProperties.ResourcePrefix,
		requestOnCreateDb.DbName)
	expectedUrl := fmt.Sprintf("http://localhost:9200/%s", expectedIndexName)
	assert.Equal(t, expectedIndexName, response.Name)
	assert.Equal(t, expectedIndexName, response.ConnectionProperties.DbName)
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Empty(t, response.ConnectionProperties.Username)
	assert.Empty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.IndexKind, Name: expectedIndexName},
		{Kind: common.MetadataKind, Name: expectedIndexName},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateIndexWithPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"index"},
		},
	}
	r, err := baseProvider.createDatabase(requestOnCreateDb, ctx)
	assert.NoError(t, err, "failed to create database")
	response, ok := r.(DbCreateResponse)
	assert.True(t, ok, "failed to cast type DbCreateResponse")
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.NotEmpty(t, response.ConnectionProperties.ResourcePrefix)
	expectedIndexName := fmt.Sprintf("%s_%s", response.ConnectionProperties.ResourcePrefix,
		requestOnCreateDb.DbName)
	expectedUrl := fmt.Sprintf("http://localhost:9200/%s", expectedIndexName)
	assert.Equal(t, expectedIndexName, response.Name)
	assert.Equal(t, expectedIndexName, response.ConnectionProperties.DbName)
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Empty(t, response.ConnectionProperties.Username)
	assert.Empty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.IndexKind, Name: expectedIndexName},
		{Kind: common.MetadataKind, Name: expectedIndexName},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateIndexWithoutPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: false,
			CreateOnly:     []string{"index"},
		},
	}
	r, err := baseProvider.createDatabase(requestOnCreateDb, ctx)
	assert.NoError(t, err, "failed to create database")
	response, ok := r.(DbCreateResponse)
	assert.True(t, ok, "failed to cast type DbCreateResponse")
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Empty(t, response.ConnectionProperties.ResourcePrefix)
	expectedIndexName := fmt.Sprintf("dbaas_%s", requestOnCreateDb.DbName)
	expectedUrl := fmt.Sprintf("http://localhost:9200/%s", expectedIndexName)
	assert.Equal(t, expectedIndexName, response.Name)
	assert.Equal(t, expectedIndexName, response.ConnectionProperties.DbName)
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Empty(t, response.ConnectionProperties.Username)
	assert.Empty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.IndexKind, Name: expectedIndexName},
		{Kind: common.MetadataKind, Name: expectedIndexName},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateUserWithPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"user"},
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.NotEmpty(t, response.ConnectionProperties.ResourcePrefix)
	assert.Empty(t, response.Name)
	assert.Empty(t, response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, response.ConnectionProperties.ResourcePrefix)
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
		{Kind: common.MetadataKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateUserWithoutPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: false,
			CreateOnly:     []string{"user"},
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Empty(t, response.ConnectionProperties.ResourcePrefix)
	assert.Empty(t, response.Name)
	assert.Empty(t, response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, "dbaas")
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.MetadataKind, Name: "dbaas"},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateIndexAndUserWithPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"index", "user"},
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.NotEmpty(t, response.ConnectionProperties.ResourcePrefix)
	expectedIndexName := fmt.Sprintf("%s_%s", response.ConnectionProperties.ResourcePrefix,
		requestOnCreateDb.DbName)
	expectedUrl := fmt.Sprintf("http://localhost:9200/%s", expectedIndexName)
	assert.Equal(t, expectedIndexName, response.Name)
	assert.Equal(t, expectedIndexName, response.ConnectionProperties.DbName)
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, response.ConnectionProperties.ResourcePrefix)
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.IndexKind, Name: expectedIndexName},
		{Kind: common.MetadataKind, Name: expectedIndexName},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateIndexAndUserWithoutPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: false,
			CreateOnly:     []string{"index", "user"},
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Empty(t, response.ConnectionProperties.ResourcePrefix)
	expectedIndexName := fmt.Sprintf("dbaas_%s", requestOnCreateDb.DbName)
	expectedUrl := fmt.Sprintf("http://localhost:9200/%s", expectedIndexName)
	assert.Equal(t, expectedIndexName, response.Name)
	assert.Equal(t, expectedIndexName, response.ConnectionProperties.DbName)
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, "dbaas")
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.IndexKind, Name: expectedIndexName},
		{Kind: common.MetadataKind, Name: expectedIndexName},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateResourcesWithoutCreateOnly(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: true,
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.NotEmpty(t, response.ConnectionProperties.ResourcePrefix)
	expectedIndexName := fmt.Sprintf("%s_%s", response.ConnectionProperties.ResourcePrefix,
		requestOnCreateDb.DbName)
	expectedUrl := fmt.Sprintf("http://localhost:9200/%s", expectedIndexName)
	assert.Equal(t, expectedIndexName, response.Name)
	assert.Equal(t, expectedIndexName, response.ConnectionProperties.DbName)
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, response.ConnectionProperties.ResourcePrefix)
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.IndexKind, Name: expectedIndexName},
		{Kind: common.MetadataKind, Name: expectedIndexName},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateResourcesWithoutCreateOnlyAndPrefix(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: nil,
		DbName:   "index",
		Settings: Settings{
			ResourcePrefix: false,
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	logger.InfoContext(ctx, fmt.Sprintf("Response is %v", response))
	assert.Empty(t, response.ConnectionProperties.ResourcePrefix)
	expectedIndexName := fmt.Sprintf("dbaas_%s", requestOnCreateDb.DbName)
	expectedUrl := fmt.Sprintf("http://localhost:9200/%s", expectedIndexName)
	assert.Equal(t, expectedIndexName, response.Name)
	assert.Equal(t, expectedIndexName, response.ConnectionProperties.DbName)
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, "dbaas")
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.IndexKind, Name: expectedIndexName},
		{Kind: common.MetadataKind, Name: expectedIndexName},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestDeleteResources(t *testing.T) {
	resources := []dao.DbResource{
		{
			Kind: common.IndexKind,
			Name: "test_index",
		},
		{
			Kind: common.MetadataKind,
			Name: "test_index",
		},
		{
			Kind: common.UserKind,
			Name: "test_user",
		},
	}
	deletedResources := baseProvider.deleteResources(resources, context.Background())
	expectedDeletedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: "test_user", Status: DeletedStatus, ErrorMessage: ""},
		{Kind: common.IndexKind, Name: "test_index", Status: DeletedStatus, ErrorMessage: ""},
		{Kind: common.MetadataKind, Name: "test_index", Status: DeletedStatus, ErrorMessage: ""},
	}
	assert.Equal(t, expectedDeletedResources, deletedResources)
}

func TestDeleteResourcesByPrefix(t *testing.T) {
	resources := []dao.DbResource{
		{
			Kind: common.ResourcePrefixKind,
			Name: "test",
		},
	}
	deletedResources := baseProvider.deleteResources(resources, context.Background())
	expectedDeletedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: "test", Status: DeletedStatus, ErrorMessage: ""},
		{Kind: common.IndexKind, Name: "test*", Status: DeletedStatus, ErrorMessage: ""},
		{Kind: common.MetadataKind, Name: "test", Status: DeletedStatus, ErrorMessage: ""},
		{Kind: common.TemplateKind, Name: "test*", Status: DeletedStatus, ErrorMessage: ""},
		{Kind: common.IndexTemplateKind, Name: "test*", Status: DeletedStatus, ErrorMessage: ""},
		{Kind: common.AliasKind, Name: "test*", Status: DeletedStatus, ErrorMessage: ""},
	}
	assert.Equal(t, expectedDeletedResources, deletedResources)
}

func TestCreateUserWithMicroserviceAndNamespaceInMetadata(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: map[string]interface{}{
			"classifier": map[string]interface{}{
				"namespace": "new-namespace",
			},
			"microserviceName": "new-microservice-name",
		},
		DbName: "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"user"},
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	assert.NotEmpty(t, response.ConnectionProperties.ResourcePrefix)
	fmt.Println(response.ConnectionProperties.ResourcePrefix)
	assert.Contains(t, response.ConnectionProperties.ResourcePrefix, "new-microservice-name_new-namespace")
	assert.Empty(t, response.Name)
	assert.Empty(t, response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, response.ConnectionProperties.ResourcePrefix)
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
		{Kind: common.MetadataKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateUserWithMicroserviceInMetadata(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: map[string]interface{}{
			"microserviceName": "new-microservice-name",
		},
		DbName: "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"user"},
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	assert.NotEmpty(t, response.ConnectionProperties.ResourcePrefix)
	assert.NotContains(t, response.ConnectionProperties.ResourcePrefix, "new-microservice-name")
	assert.Empty(t, response.Name)
	assert.Empty(t, response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, response.ConnectionProperties.ResourcePrefix)
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
		{Kind: common.MetadataKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}

func TestCreateUserWithNamespaceInMetadata(t *testing.T) {
	requestOnCreateDb := DbCreateRequest{
		Metadata: map[string]interface{}{
			"classifier": map[string]interface{}{
				"namespace": "new-namespace",
			},
		},
		DbName: "index",
		Settings: Settings{
			ResourcePrefix: true,
			CreateOnly:     []string{"user"},
		},
	}
	r, _ := baseProvider.createDatabase(requestOnCreateDb, ctx)
	response := r.(DbCreateResponse)
	assert.NotEmpty(t, response.ConnectionProperties.ResourcePrefix)
	assert.NotContains(t, response.ConnectionProperties.ResourcePrefix, "new-namespace")
	assert.Empty(t, response.Name)
	assert.Empty(t, response.ConnectionProperties.DbName)
	expectedUrl := "http://localhost:9200/"
	assert.Equal(t, expectedUrl, response.ConnectionProperties.Url)
	assert.Contains(t, response.ConnectionProperties.Username, response.ConnectionProperties.ResourcePrefix)
	assert.NotEmpty(t, response.ConnectionProperties.Password)
	expectedResources := []dao.DbResource{
		{Kind: common.UserKind, Name: response.ConnectionProperties.Username},
		{Kind: common.ResourcePrefixKind, Name: response.ConnectionProperties.ResourcePrefix},
		{Kind: common.MetadataKind, Name: response.ConnectionProperties.ResourcePrefix},
	}
	assert.ElementsMatch(t, response.Resources, expectedResources)
}
