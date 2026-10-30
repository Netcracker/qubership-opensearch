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
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/Netcracker/dbaas-opensearch-adapter/basic"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"github.com/Netcracker/qubership-dbaas-adapter-core/pkg/dao"
)

// DescriptorPath is the path of the endpoint that is used by DBaaS Operator to get the physical database descriptor.
const DescriptorPath = "/api/v2/adapter/physical_database"

const physicalDatabaseType = "opensearch"

// Descriptor describes the physical database in the format expected by DBaaS Operator.
type Descriptor struct {
	PhysicalDatabaseId string            `json:"physicalDatabaseId"`
	Type               string            `json:"type"`
	Labels             map[string]string `json:"labels"`
	ApiVersions        dao.ApiVersions   `json:"apiVersions"`
	Features           map[string]bool   `json:"features"`
	SupportedRoles     []string          `json:"supportedRoles"`
	ReadOnlyHost       string            `json:"readOnlyHost,omitempty"`
}

// BuildDescriptor prepares the physical database descriptor. It does not interact with DBaaS aggregator.
func BuildDescriptor(physicalDatabaseId string, labels map[string]string, baseProvider *basic.BaseProvider) (*Descriptor, error) {
	if physicalDatabaseId == "" {
		return nil, fmt.Errorf("physical database identifier is not specified")
	}
	if labels == nil {
		labels = map[string]string{}
	}
	return &Descriptor{
		PhysicalDatabaseId: physicalDatabaseId,
		Type:               physicalDatabaseType,
		Labels:             labels,
		ApiVersions: dao.ApiVersions{Specs: []dao.ApiVersionsSpec{
			{
				SpecRootUrl:     dao.RootUrl,
				Major:           dao.MajorAPIVersion,
				Minor:           dao.MinorAPIVersion,
				SupportedMajors: dao.SupportedMajorsVersions,
			},
		}},
		Features: map[string]bool{
			"multiusers": true,
			"tls":        baseProvider.IsOpenSearchTlsEnabled(),
		},
		SupportedRoles: baseProvider.GetSupportedRoleTypes(),
	}, nil
}

// DescriptorHandler returns the handler for the physical database descriptor endpoint.
// The handler never performs registration in DBaaS aggregator, it is done by DBaaS Operator.
func DescriptorHandler(physicalDatabaseId string, labelsFileLocation string,
	baseProvider *basic.BaseProvider) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		logger.InfoContext(ctx, "Received request to get physical database descriptor")
		descriptor, err := BuildDescriptor(physicalDatabaseId, readLabels(labelsFileLocation), baseProvider)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to build physical database descriptor", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusServiceUnavailable)
			return
		}
		responseBody, err := json.Marshal(descriptor)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to marshal physical database descriptor to json", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		common.ProcessResponseBody(ctx, w, responseBody, 0)
	}
}

func readLabels(labelsFileLocation string) map[string]string {
	file, err := os.ReadFile(labelsFileLocation)
	if err != nil {
		logger.Warn(fmt.Sprintf("Skipping labels file, cannot read it: %s", labelsFileLocation))
		return nil
	}
	var labels map[string]string
	if err = json.Unmarshal(file, &labels); err != nil {
		logger.Warn(fmt.Sprintf("Failed to parse labels file %s", labelsFileLocation), slog.Any("error", err))
	}
	return labels
}
