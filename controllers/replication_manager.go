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
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Netcracker/opensearch-service/util"
	"github.com/go-logr/logr"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const (
	leaderAlias                    = "leader-cluster"
	startFullReplicationPath       = "_plugins/_replication/_autofollow"
	indexReplicationStatusPattern  = "_plugins/_replication/%s/_status"
	replicationName                = "dr-replication"
	replicationNotInProgressStatus = "REPLICATION NOT IN PROGRESS"
	replicationAttemptsNumber      = 5
	replicationCheckTimeout        = time.Second * 15
)

type ReplicationManager struct {
	restClient util.RestClient
	remoteUrl  string
	pattern    string
	logger     logr.Logger
}

/*
ReplicationStats is a struct to unmarshal http response
The following json is expected as response:

	{
		"num_syncing_indices": 3,
		"num_bootstrapping_indices": 0,
		"num_paused_indices": 0,
		"num_failed_indices": 0,
		"num_shard_tasks": 3,
		"num_index_tasks": 3,
		"operations_written": 6,
		"operations_read": 6,
		"failed_read_requests": 0,
		"throttled_read_requests": 0,
		"failed_write_requests": 0,
		"throttled_write_requests": 0,
		"follower_checkpoint": 3,
		"leader_checkpoint": 5,
		"total_write_time_millis": 194,
		"index_stats": {
			"my-demo45": {
				"operations_written": 0,
				"operations_read": 0,
				"failed_read_requests": 0,
				"throttled_read_requests": 0,
				"failed_write_requests": 0,
				"throttled_write_requests": 0,
				"follower_checkpoint": -1,
				"leader_checkpoint": 0,
				"total_write_time_millis": 0
			},
			"my-demo": {
				"operations_written": 0,
				"operations_read": 0,
				"failed_read_requests": 0,
				"throttled_read_requests": 0,
				"failed_write_requests": 0,
				"throttled_write_requests": 0,
				"follower_checkpoint": -1,
				"leader_checkpoint": 0,
				"total_write_time_millis": 0
			},
			"my-index1": {
				"operations_written": 6,
				"operations_read": 6,
				"failed_read_requests": 0,
				"throttled_read_requests": 0,
				"failed_write_requests": 0,
				"throttled_write_requests": 0,
				"follower_checkpoint": 5,
				"leader_checkpoint": 5,
				"total_write_time_millis": 194
			}
		}
	}
*/
type ReplicationStats struct {
	SyncingIndicesCount       int                 `json:"num_syncing_indices"`
	BootstrappingIndicesCount int                 `json:"num_bootstrapping_indices"`
	PausedIndicesCount        int                 `json:"num_paused_indices"`
	FailedIndicesCount        int                 `json:"num_failed_indices"`
	IndexStats                map[string]struct{} `json:"index_stats"`
}

type NodesInfo struct {
	Nodes map[string]TasksInfo `json:"nodes"`
}

type TasksInfo struct {
	Tasks map[string]InnerTask `json:"tasks"`
}

type InnerTask struct {
	Action string `json:"action"`
}

type RuleStats struct {
	Name          string   `json:"name"`
	Pattern       string   `json:"pattern"`
	SuccessStart  int      `json:"num_success_start_replication"`
	FailedStart   int      `json:"num_failed_start_replication"`
	FailedIndices []string `json:"failed_indices"`
}

type AutofollowStats struct {
	SuccessStart        int         `json:"num_success_start_replication"`
	FailedStart         int         `json:"num_failed_start_replication"`
	AutofollowRuleStats []RuleStats `json:"autofollow_stats"`
}

type ReplicationIndexStats struct {
	Status  string       `json:"status"`
	Reason  string       `json:"reason,omitempty"`
	Details IndexDetails `json:"syncing_details,omitempty"`
}

type IndexDetails struct {
	LeaderCheckpoint   int `json:"leader_checkpoint"`
	FollowerCheckpoint int `json:"follower_checkpoint"`
	Seq                int `json:"seq_no"`
}

type PluginReplicationError struct {
	Err struct {
		Reason   string `json:"reason,omitempty"`
		Type     string `json:"type,omitempty"`
		CausedBy struct {
			Type   string `json:"type,omitempty"`
			Reason string `json:"reason,omitempty"`
		} `json:"caused_by,omitempty"`
	} `json:"error"`
	Status int `json:"status"`
}

func NewReplicationManager(restClient util.RestClient, remoteUrl string, indexPattern string, logger logr.Logger) *ReplicationManager {
	return &ReplicationManager{
		restClient: restClient,
		remoteUrl:  remoteUrl,
		pattern:    indexPattern,
		logger:     logger,
	}
}

func (rm ReplicationManager) Configure() error {
	path := "_cluster/settings"
	body := fmt.Sprintf(`
{
  "persistent": {
    "cluster": {
	  "remote": {
		"%s": {
		  "seeds": ["%s"]
		}
	  }
    }
  }
}
`, leaderAlias, rm.remoteUrl)
	statusCode, _, err := rm.restClient.SendRequest(http.MethodPut, path, strings.NewReader(body))
	if err != nil {
		return err
	}
	if statusCode >= 400 {
		return fmt.Errorf("asynchronous request to create connection with the remote opensearch cluster returned unexpected status code - [%d]",
			statusCode)
	}
	return nil
}

func (rm ReplicationManager) Start() error {
	body := fmt.Sprintf(`
{
  "leader_alias": "%s",
  "pattern": "%s",
  "name": "%s",
  "use_roles": {
    "leader_cluster_role": "all_access",
	"follower_cluster_role": "all_access"
  }
}
`, leaderAlias, rm.pattern, replicationName)
	statusCode, respBody, err := rm.restClient.SendRequest(http.MethodPost, startFullReplicationPath, strings.NewReader(body))
	if err != nil {
		return err
	}
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}
	var replicationErrorData PluginReplicationError
	err = json.Unmarshal(respBody, &replicationErrorData)
	if err != nil {
		return err
	}
	if statusCode == 500 && replicationErrorData.Err.Type == "connect_transport_exception" {
		if replicationErrorData.Err.CausedBy.Reason == "handshake failed because connection reset" {
			return fmt.Errorf("leader and follower opensearch clusters have different admin and transport certs, the full error message is %+v", replicationErrorData)
		}
		if strings.HasPrefix(replicationErrorData.Err.CausedBy.Reason, "Connection refused") {
			return fmt.Errorf("the leader opensearch cluster was resolved but connection refused, please, check leader cluster configuration carefully. The full error message is %+v", replicationErrorData)
		}
	}
	if statusCode == 404 && replicationErrorData.Err.Type == "no_such_remote_cluster_exception" {
		return fmt.Errorf("the opensearch cluster with alias [%s] was not found, check that it is created on the previous step. The full error message is %+v", leaderAlias, replicationErrorData)
	}
	if statusCode == 400 &&
		replicationErrorData.Err.Type == "illegal_argument_exception" &&
		replicationErrorData.Err.CausedBy.Type == "unknown_host_exception" {
		return fmt.Errorf("remote opensearch cluster host - [%s] is invalid. The full error message is %+v", replicationErrorData.Err.CausedBy.Reason, replicationErrorData)
	}
	return fmt.Errorf("error occurred during start dr replication - [%v]", replicationErrorData)
}

func (rm ReplicationManager) RemoveReplicationRule() error {
	rule, err := rm.GetAutoFollowRuleStats()
	if err != nil {
		return fmt.Errorf(fmt.Sprintf("failed to get replication rule, reason: %e", err))
	}
	if rule == nil {
		rm.logger.Info("Skipping replication rule removal since its does not exist")
		return nil
	}
	body := fmt.Sprintf(`{"leader_alias": "%s","name": "%s"}`, leaderAlias, replicationName)
	statusCode, _, err := rm.restClient.SendRequest(http.MethodDelete, startFullReplicationPath, strings.NewReader(body))
	if err != nil {
		return err
	}
	if statusCode >= 400 && statusCode != http.StatusNotFound {
		return fmt.Errorf("internal server error with %d status code", statusCode)
	}
	return nil
}

func (rm ReplicationManager) StopReplication() error {
	indexNames, err := rm.getReplicatedIndices()
	if err != nil {
		return err
	}
	if len(indexNames) == 0 {
		return nil
	}

	if err = rm.stopIndicesReplication(indexNames); err != nil {
		return err
	}
	return nil
}

func (rm ReplicationManager) getReplicatedIndices() ([]string, error) {
	statusCode, resp, err := rm.restClient.SendRequest(http.MethodGet, "_plugins/_replication/follower_stats", nil)
	if err != nil {
		return nil, err
	}
	if statusCode >= 400 {
		return nil, errors.New("internal server error")
	}
	var replicationStats ReplicationStats
	if err = json.Unmarshal(resp, &replicationStats); err != nil {
		return nil, err
	}
	var indices []string
	for key := range replicationStats.IndexStats {
		indices = append(indices, key)
	}
	return indices, nil
}

func (rm ReplicationManager) executeReplicationCheck(indexNames []string) error {
	//TODO: should we execute replication health check here?
	inProgressIndices := make(map[string]int)
	var failedIndices []string
	pattern := strings.ReplaceAll(rm.pattern, "*", ".*")
	for _, index := range indexNames {
		matched, err := regexp.MatchString(pattern, index)
		if err != nil {
			rm.logger.Error(err, fmt.Sprintf("Regular expression - [%s] is invalid", rm.pattern))
			return err
		}
		if !matched {
			continue
		}
		replicationIndexStats, err := rm.getIndexReplicationStatus(index)
		if err != nil {
			rm.logger.Error(err, fmt.Sprintf("Can not get replication index stats for [%s] index", index))
			return err
		}
		if replicationIndexStats.Status == "SYNCING" || replicationIndexStats.Status == "BOOTSTRAPPING" {
			if replicationIndexStats.Details.LeaderCheckpoint != replicationIndexStats.Details.FollowerCheckpoint {
				inProgressIndices[index] = replicationIndexStats.Details.LeaderCheckpoint
			}
		} else {
			failedIndices = append(failedIndices, index)
		}
	}

	if len(failedIndices) > 0 {
		err := fmt.Errorf("some replication indices are failed")
		rm.logger.Error(err, fmt.Sprintf("Replication check is failed because there are failed replication indices: [%v]", failedIndices))
		return err
	}

	attempts := replicationAttemptsNumber
	for attempts > 0 {
		restProgressIndex, err := rm.updateInProgressIndices(inProgressIndices)
		if err != nil {
			rm.logger.Error(err, "Can not update list of in progress replication indices.")
			return err
		}
		if len(restProgressIndex) == 0 {
			rm.logger.Info("Replication check is done")
			return nil
		}
		rm.logger.Info(fmt.Sprintf("The rest replicated indices in progress are [%v]", restProgressIndex))
		attempts -= 1
		time.Sleep(replicationCheckTimeout)
	}
	return fmt.Errorf("replication check was failed after 5 attempts")
}

func (rm ReplicationManager) getIndexReplicationStatus(index string) (ReplicationIndexStats, error) {
	path := fmt.Sprintf(indexReplicationStatusPattern, index)
	replicationIndexStats, err := rm.getReplicationIndexStats(path)
	if err != nil {
		rm.logger.Error(err, fmt.Sprintf("Can not get replication index stats for [%s] index", index))
		return ReplicationIndexStats{}, err
	}
	return replicationIndexStats, nil
}

func (rm ReplicationManager) getReplicationIndexStats(path string) (ReplicationIndexStats, error) {
	_, body, err := rm.restClient.SendRequest(http.MethodGet, path, nil)
	if err != nil {
		fmt.Println("replication index stats error")
		return ReplicationIndexStats{}, err
	}
	var replicationIndexStats ReplicationIndexStats
	err = json.Unmarshal(body, &replicationIndexStats)
	if err != nil {
		fmt.Println("unmarshal error per index")
		return ReplicationIndexStats{}, err
	}
	return replicationIndexStats, nil
}

func (rm ReplicationManager) updateInProgressIndices(inProgressIndices map[string]int) (map[string]int, error) {
	pathTemplate := "_plugins/_replication/%s/_status"
	restProgressIndex := make(map[string]int)
	for index, leaderCheckpoint := range inProgressIndices {
		path := fmt.Sprintf(pathTemplate, index)
		replicationIndexStats, err := rm.getReplicationIndexStats(path)
		if err != nil {
			rm.logger.Error(err, fmt.Sprintf("can not get replication index ([%s]) statistic to update list of in progress indices", index))
			return nil, err
		}
		if replicationIndexStats.Details.FollowerCheckpoint < leaderCheckpoint {
			restProgressIndex[index] = leaderCheckpoint
		}
	}
	return restProgressIndex, nil
}

func (rm ReplicationManager) stopIndicesReplication(indexNames []string) error {
	stopIndexReplicationTemplate := "_plugins/_replication/%s/_stop"
	for _, index := range indexNames {
		statusCode, responseBody, err :=
			rm.restClient.SendRequest(http.MethodPost,
				fmt.Sprintf(stopIndexReplicationTemplate, index),
				strings.NewReader(`{}`))
		if err != nil {
			return err
		}
		if statusCode >= 400 {
			return fmt.Errorf("can not stop replication for [%s] index with status code - [%d], response - [%s]",
				index, statusCode, string(responseBody))
		}
		rm.logger.Info(fmt.Sprintf("Replication was stopped for index [%s]", index))
	}
	//TODO: Check that Replication is stopped
	time.Sleep(time.Second * 3)
	return nil
}

func (rm ReplicationManager) DeleteIndices() error {
	if rm.pattern != "*" {
		if err := rm.DeleteIndicesByPatternWithUnlock(rm.pattern); err != nil {
			return err
		}
		return nil
	}

	indices, err := rm.restClient.GetArrayData("_cat/indices?h=index", "index", func(s string) bool {
		return !strings.HasPrefix(s, ".")
	})
	if err != nil {
		return err
	}

	for _, index := range indices {
		if err = rm.DeleteIndicesByPatternWithUnlock(index); err != nil {
			return err
		}
	}
	return nil
}

func (rm ReplicationManager) StopIndicesReplicationByPattern(pattern string) error {
	path := fmt.Sprintf("_cat/indices/%s?h=index", pattern)
	indices, err := rm.restClient.GetArrayData(path, "index", func(index string) bool {
		if strings.HasPrefix(index, ".") {
			return false
		}
		replicationStatus, err := rm.getIndexReplicationStatus(index)
		if err != nil {
			log.Error(err, fmt.Sprintf("Cannot get replication status of [%s] index, skip it", index))
			return false
		}
		return replicationStatus.Status != replicationNotInProgressStatus
	})
	if err != nil {
		return err
	}
	return rm.stopIndicesReplication(indices)
}

func (rm ReplicationManager) DeleteIndicesByPatternWithUnlock(pattern string) error {
	statusCode, _, err := rm.restClient.SendRequest(http.MethodDelete, pattern, nil)
	if err != nil {
		return err
	}
	if statusCode == 403 {
		indices, err := rm.GetIndicesByPatternExcludeService(pattern)
		if err != nil {
			return err
		}
		if err = rm.stopIndicesReplication(indices); err != nil {
			return err
		}
		if err = rm.DeleteIndicesByPattern(pattern); err != nil {
			return err
		}
	}
	if statusCode >= 400 {
		return fmt.Errorf("can not delete indices by pattern - [%s] with status code - [%d]",
			pattern, statusCode)
	}
	return nil
}

func (rm ReplicationManager) GetIndicesByPatternExcludeService(pattern string) ([]string, error) {
	path := fmt.Sprintf("_cat/indices/%s?h=index", pattern)
	indices, err := rm.restClient.GetArrayData(path, "index", func(s string) bool {
		return !strings.HasPrefix(s, ".")
	})
	return indices, err
}

func (rm ReplicationManager) DeleteIndicesByPattern(pattern string) error {
	statusCode, _, err := rm.restClient.SendRequest(http.MethodDelete, pattern, nil)
	if err != nil {
		return err
	}
	if statusCode == 403 {
		rm.logger.Info(fmt.Sprintf("No permissions to delete Opensearch indices by pattern: [%s]", pattern))
		return errors.New("no permissions to delete opensearch indices by pattern")
	}
	if statusCode >= 400 {
		return fmt.Errorf("can not delete indices by [%s] pattern with [%d] status code", pattern, statusCode)
	}
	return nil
}

func (rm ReplicationManager) DeleteAdminReplicationTasks() error {
	_, body, err := rm.restClient.SendRequest(http.MethodGet, "_tasks", nil)
	if err != nil {
		rm.logger.Error(err, "admin replication task was not deleted")
		return err
	}
	var nodes NodesInfo
	err = json.Unmarshal(body, &nodes)
	if err != nil {
		rm.logger.Error(err, "admin replication task was not deleted")
		return err
	}
	var tasksToDelete []string
	for _, tasks := range nodes.Nodes {
		for taskId, innerTask := range tasks.Tasks {
			if innerTask.Action == "cluster:indices/admin/replication[c]" {
				tasksToDelete = append(tasksToDelete, taskId)
			}
		}
	}

	rm.logger.Info(fmt.Sprintf("admin replication tasks are found - %v", tasksToDelete))
	for _, task := range tasksToDelete {
		path := fmt.Sprintf("_tasks/%s/_cancel", task)
		_, _, err := rm.restClient.SendRequest(http.MethodPost, path, nil)
		if err != nil {
			rm.logger.Error(err, "admin replication task was not deleted")
			return err
		}
	}
	rm.logger.Info("admin replication tasks were deleted")
	return nil
}

func (rm ReplicationManager) GetAutoFollowRuleStats() (*RuleStats, error) {
	_, body, err := rm.restClient.SendRequest(http.MethodGet, "_plugins/_replication/autofollow_stats", nil)
	if err != nil {
		rm.logger.Error(err, "unable to read autofollow statistic")
		return nil, err
	}
	var stats AutofollowStats
	err = json.Unmarshal(body, &stats)
	if err != nil {
		rm.logger.Error(err, "unable to unmarshal autofollow statistic")
		return nil, err
	}
	for _, rule := range stats.AutofollowRuleStats {
		if rule.Name == replicationName {
			return &rule, nil
		}
	}
	rm.logger.Info("Unable to find existing DR replication rule")
	return nil, nil
}
