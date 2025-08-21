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
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Netcracker/dbaas-opensearch-adapter/common"
)

const (
	RecoveryIdleState    = "idle"
	RecoveryRunningState = "running"
	RecoveryFailedState  = "failed"
	RecoveryDoneState    = "done"
	batchSize            = 100
)

type UsersToRecover struct {
	Settings             map[string]interface{}        `json:"settings,omitempty"`
	ConnectionProperties []common.ConnectionProperties `json:"connectionProperties"`
}

func (bp *BaseProvider) RecoverUsersHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := common.PrepareContext(r)
		decoder := json.NewDecoder(r.Body)
		var usersToRecover UsersToRecover
		err := decoder.Decode(&usersToRecover)
		if err != nil {
			logger.ErrorContext(ctx, "Failed to decode request in recover users handler", slog.Any("error", err))
			common.ProcessResponseBody(ctx, w, []byte(err.Error()), http.StatusInternalServerError)
			return
		}
		if bp.recoveryState != RecoveryRunningState {
			bp.recoveryState = RecoveryRunningState
			go bp.recovery(usersToRecover.ConnectionProperties, ctx)
		}
		w.WriteHeader(http.StatusOK)
	}
}

func (bp *BaseProvider) GetRecoveryStateHandler() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		responseBody := []byte(bp.recoveryState)
		common.ProcessResponseBody(ctx, w, responseBody, http.StatusOK)
	}
}

func (bp *BaseProvider) recovery(connectionProperties []common.ConnectionProperties, ctx context.Context) {
	var changes []Change
	for _, properties := range connectionProperties {
		changes = append(changes, Change{
			Operation: "add",
			Path:      fmt.Sprintf("/%s", properties.Username),
			Value:     bp.getUserContent(properties),
		})
	}
	position := 0
	var batch []Change
	for position < len(changes) {
		if position+batchSize < len(changes) {
			batch = changes[position : position+batchSize]
		} else {
			batch = changes[position:]
		}
		logger.DebugContext(ctx, fmt.Sprintf("Current batch size is %d", len(batch)))
		var err error
		// 3 attempts to create corresponding patch of users
		for i := 0; i < 3; i++ {
			err = bp.patchUsers(batch, ctx)
			if err == nil {
				break
			}
			time.Sleep(10 * time.Second)
		}
		if err != nil {
			bp.recoveryState = RecoveryFailedState
			logger.ErrorContext(ctx, fmt.Sprintf("Unable to restore users because of error: %+v", err))
			return
		}
		position += batchSize
	}
	bp.recoveryState = RecoveryDoneState
	logger.InfoContext(ctx, "Users recovery is successfully finished")
}

func (bp *BaseProvider) getUserContent(properties common.ConnectionProperties) Content {
	roleType := AdminRoleType
	if properties.Role != "" {
		roleType = properties.Role
	}
	prefix := properties.ResourcePrefix
	if prefix == "" {
		prefix = properties.DbName
	}
	return Content{
		Attributes:   map[string]string{resourcePrefixAttributeName: prefix},
		BackendRoles: bp.GetBackendRoles(roleType),
		Password:     properties.Password,
	}
}
