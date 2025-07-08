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
	"fmt"
	"github.com/Netcracker/opensearch-service/operator/util"
	"github.com/go-logr/logr"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	allIndicesExceptSystemPattern      = "*,-.*"
	indicesExceptSystemPatternTemplate = "%s,-.*"
	runningWatcherState                = "running"
	stoppedWatcherState                = "stopped"
	watchInterval                      = 60 * time.Second
)

type SlowLogIndicesHelper struct {
	logger     logr.Logger
	restClient *util.RestClient
}

type SlowLogIndicesWatcher struct {
	lock  *sync.Mutex
	State *string
}

func NewSlowLogIndicesWatcher(mutex *sync.Mutex) SlowLogIndicesWatcher {
	state := stoppedWatcherState
	return SlowLogIndicesWatcher{
		lock:  mutex,
		State: &state,
	}
}

func (sliw SlowLogIndicesWatcher) start(helper SlowLogIndicesHelper, indicesPattern string, minSeconds int) {
	sliw.stop(helper)
	*sliw.State = runningWatcherState
	go sliw.watch(helper, indicesPattern, minSeconds)
}

func (sliw SlowLogIndicesWatcher) stop(helper SlowLogIndicesHelper) {
	if *sliw.State != stoppedWatcherState {
		*sliw.State = stoppedWatcherState
		sliw.removeSlowLogSetting(helper)
	}
}

func (sliw SlowLogIndicesWatcher) watch(helper SlowLogIndicesHelper, indicesPattern string, minSeconds int) {
	sliw.lock.Lock()
	defer sliw.lock.Unlock()
	for {
		if *sliw.State == stoppedWatcherState {
			helper.logger.Info("SlowLog Indices Watcher is stopped, exit from watch loop")
			return
		}
		sliw.addSlowLogSetting(helper, indicesPattern, minSeconds)
		time.Sleep(watchInterval)
	}
}

func (sliw SlowLogIndicesWatcher) addSlowLogSetting(helper SlowLogIndicesHelper, indicesPattern string, minSeconds int) {
	pattern := fmt.Sprintf(indicesExceptSystemPatternTemplate, indicesPattern)
	body := fmt.Sprintf(`{"search": {"slowlog": {"threshold": {"query": {"warn": "-1", "trace": "-1", "debug": "-1", "info": "%ds"}}}}}`, minSeconds)
	sliw.updateSettings(helper, pattern, body)
}

func (sliw SlowLogIndicesWatcher) removeSlowLogSetting(helper SlowLogIndicesHelper) {
	sliw.lock.Lock()
	defer sliw.lock.Unlock()
	body := `{"search": {"slowlog": {"threshold": {"query": {"warn": null, "trace": null, "debug": null, "info": null}}}}}`
	sliw.updateSettings(helper, allIndicesExceptSystemPattern, body)
}

func (sliw SlowLogIndicesWatcher) updateSettings(helper SlowLogIndicesHelper, indicesPattern string, body string) {
	path := fmt.Sprintf("%s/_settings?allow_no_indices=true", indicesPattern)
	statusCode, responseBody, err := helper.restClient.SendRequest(http.MethodPut, path, strings.NewReader(body))
	if err != nil {
		helper.logger.Error(err, "unable to update indices `slowlog` settings")
	}
	helper.logger.Info(fmt.Sprintf("Update settings request is finished with `%d` status code and body: %s",
		statusCode, string(responseBody)))
}
