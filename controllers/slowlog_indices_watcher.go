package controllers

import (
	"fmt"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
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
