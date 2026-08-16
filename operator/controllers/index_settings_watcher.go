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

package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	opensearchservice "github.com/Netcracker/qubership-opensearch/operator/api/v1"
	"github.com/Netcracker/qubership-opensearch/operator/util"
	"github.com/go-logr/logr"
	"net/http"
	"sync"
	"time"
)

type IndexSettingsHelper struct {
	logger     logr.Logger
	restClient *util.RestClient
}

type IndexSettingsWatcher struct {
	lock  *sync.Mutex
	State *string
}

func NewIndexSettingsWatcher(mutex *sync.Mutex) IndexSettingsWatcher {
	state := stoppedWatcherState
	return IndexSettingsWatcher{
		lock:  mutex,
		State: &state,
	}
}

func (isw IndexSettingsWatcher) start(helper IndexSettingsHelper, entries []opensearchservice.IndexSettingEntry) {
	isw.stop()
	*isw.State = runningWatcherState
	go isw.watch(helper, entries)
}

func (isw IndexSettingsWatcher) stop() {
	*isw.State = stoppedWatcherState
}

func (isw IndexSettingsWatcher) watch(helper IndexSettingsHelper, entries []opensearchservice.IndexSettingEntry) {
	isw.lock.Lock()
	defer isw.lock.Unlock()
	for {
		if *isw.State == stoppedWatcherState {
			helper.logger.Info("Index Settings Watcher is stopped, exit from watch loop")
			return
		}
		isw.applyAllSettings(helper, entries)
		time.Sleep(watchInterval)
	}
}

func (isw IndexSettingsWatcher) applyAllSettings(helper IndexSettingsHelper, entries []opensearchservice.IndexSettingEntry) {
	for _, entry := range entries {
		body, err := json.Marshal(entry.Settings)
		if err != nil {
			helper.logger.Error(err, "unable to serialize index settings", "pattern", entry.Pattern)
			continue
		}
		pattern := fmt.Sprintf(indicesExceptSystemPatternTemplate, entry.Pattern)
		path := fmt.Sprintf("%s/_settings?allow_no_indices=true", pattern)
		statusCode, responseBody, err := helper.restClient.SendRequest(http.MethodPut, path, bytes.NewReader(body))
		if err != nil {
			helper.logger.Error(err, "unable to apply index settings", "pattern", entry.Pattern)
			continue
		}
		helper.logger.V(1).Info(fmt.Sprintf("Applied index settings for pattern '%s': status %d, response: %s",
			entry.Pattern, statusCode, string(responseBody)))
	}
}
