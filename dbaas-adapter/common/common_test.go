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

package common

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConvertConnectionPropertiesToMap(t *testing.T) {
	connectionProperties := ConnectionProperties{
		DbName:         "",
		Host:           "opensearch.opensearch-service",
		Port:           9200,
		Url:            "http://opensearch.opensearch-service:9200/",
		Username:       "a73a026a-da44-4257-aed8-b1ee1bff5b7c",
		Password:       "WPJdruZ9@F",
		ResourcePrefix: "a73a026a-da44-4257-aed8-b1ee1bff5b7c",
		Role:           "dml",
		Tls:            false,
	}
	actualMap, err := ConvertStructToMap(connectionProperties)
	expectedMap := map[string]interface{}{
		"dbName":         "",
		"host":           "opensearch.opensearch-service",
		"port":           float64(9200),
		"url":            "http://opensearch.opensearch-service:9200/",
		"username":       "a73a026a-da44-4257-aed8-b1ee1bff5b7c",
		"password":       "WPJdruZ9@F",
		"resourcePrefix": "a73a026a-da44-4257-aed8-b1ee1bff5b7c",
		"role":           "dml",
	}
	assert.Empty(t, err)
	assert.EqualValues(t, expectedMap, actualMap)
}

func TestConvertEmptyConnectionPropertiesToMap(t *testing.T) {
	connectionProperties := ConnectionProperties{}
	actualMap, err := ConvertStructToMap(connectionProperties)
	expectedMap := map[string]interface{}{
		"dbName": "",
		"host":   "",
		"port":   float64(0),
		"url":    "",
	}
	assert.Empty(t, err)
	assert.EqualValues(t, expectedMap, actualMap)
}
