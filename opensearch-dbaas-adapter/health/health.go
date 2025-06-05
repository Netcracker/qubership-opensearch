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

package health

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Netcracker/dbaas-opensearch-adapter/cluster"
	"github.com/Netcracker/dbaas-opensearch-adapter/common"
	"log/slog"
	"net/http"
)

type Health struct {
	Status                string                  `json:"status"`
	OpensearchHealth      common.ComponentHealth  `json:"opensearchHealth"`
	DbaasAggregatorHealth *common.ComponentHealth `json:"dbaasAggregatorHealth"`
	Opensearch            *cluster.Opensearch     `json:"-"`
}

var healthStatuses = []string{common.Down, common.OutOfService, common.Problem, common.Warning, common.Unknown, common.Up}

func (h *Health) HealthHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		h.DetermineHealthStatus(ctx)
		responseBody, err := json.Marshal(h)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			errorMessage := fmt.Sprintf("Error occurred during health serialization: %s", err.Error())
			_, err = w.Write([]byte(errorMessage))
			if err != nil {
				slog.Error("failed to write bytes to http response", slog.String("error", err.Error()), slog.String("http message", errorMessage))
			}
			return
		}
		_, err = w.Write(responseBody)
		if err != nil {
			slog.Error("failed to write bytes to http response", slog.String("error", err.Error()))
		}
	}
}

func (h *Health) DetermineHealthStatus(ctx context.Context) {
	h.OpensearchHealth.Status = h.Opensearch.GetHealth(ctx)
	for _, status := range healthStatuses {
		if status == h.OpensearchHealth.Status || status == h.DbaasAggregatorHealth.Status {
			h.Status = status
			return
		}
	}
}
