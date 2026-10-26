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

package physical

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Netcracker/dbaas-opensearch-adapter/basic"
	"github.com/stretchr/testify/assert"
)

func TestDescriptorHandler(t *testing.T) {
	handler := DescriptorHandler("test-id", "/not/existing/labels.json", &basic.BaseProvider{})
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, DescriptorPath, nil))

	assert.Equal(t, http.StatusOK, recorder.Code)
	var descriptor Descriptor
	assert.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &descriptor))
	assert.Equal(t, "test-id", descriptor.PhysicalDatabaseId)
	assert.Equal(t, "opensearch", descriptor.Type)
	assert.True(t, descriptor.Features["multiusers"])
	assert.NotEmpty(t, descriptor.SupportedRoles)
	assert.Len(t, descriptor.ApiVersions.Specs, 1)
	assert.Empty(t, descriptor.ReadOnlyHost)
}

func TestDescriptorHandlerWithoutPhysicalDatabaseId(t *testing.T) {
	handler := DescriptorHandler("", "/not/existing/labels.json", &basic.BaseProvider{})
	recorder := httptest.NewRecorder()
	handler(recorder, httptest.NewRequest(http.MethodGet, DescriptorPath, nil))

	assert.Equal(t, http.StatusServiceUnavailable, recorder.Code)
}
