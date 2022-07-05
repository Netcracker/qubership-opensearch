package disasterrecovery

import (
	"encoding/json"
	"fmt"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/controllers"
	"net/http"
	"regexp"
	"strings"
)

const (
	catIndicesPath                = "_cat/indices?h=index,health&format=json"
	indexReplicationStatusPattern = "_plugins/_replication/%s/_status"
	failedStatus                  = "FAILED"
)

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

type IndexReplicationStatus struct {
	Status string `json:"status"`
}

type Index struct {
	Index  string `json:"index"`
	Health string `json:"health"`
}

func NewReplicationChecker(opensearchName string, username string, password string) ReplicationChecker {
	var credentials []string
	if username != "" && password != "" {
		credentials = []string{username, password}
	}
	restClient := controllers.NewRestClient(createUrl(opensearchName, 9200), http.Client{}, credentials)
	return ReplicationChecker{
		restClient: *restClient,
	}
}

type ReplicationChecker struct {
	restClient controllers.RestClient
}

func (rc ReplicationChecker) checkReplication() (string, error) {
	statusCode, responseBody, err := rc.restClient.SendRequest(http.MethodGet, "_plugins/_replication/autofollow_stats", nil)
	if err != nil {
		log.Error(err, "An error occurred during autofollow_stats HTTP request")
		return "", err
	}
	if statusCode >= 500 {
		log.Error(err, "Opensearch returned status code more than 500")
		return "", fmt.Errorf("internal server error")
	}
	var autofollowStats AutofollowStats
	err = json.Unmarshal(responseBody, &autofollowStats)
	if err != nil {
		log.Error(err, "An error occurred during unmarshalling autofollow_stats HTTP response")
		return "", err
	}
	for _, rule := range autofollowStats.AutofollowRuleStats {
		if rule.Name == replicationName {
			if len(rule.FailedIndices) == 0 {
				if rule.FailedStart > 0 {
					return DEGRADED, nil
				}
			} else {
				if rule.SuccessStart > 0 {
					return DEGRADED, nil
				} else {
					return DOWN, nil
				}
			}
		}
		unhealthyIndices, err := rc.listUnhealthyIndices(rule.Pattern)
		if err != nil {
			return "", err
		}
		if len(unhealthyIndices) > 0 {
			log.Info(fmt.Sprintf("The following indices are not healthy: %v", unhealthyIndices))
			return DEGRADED, nil
		}
		failedReplications, err := rc.listFailedReplications(rule.Pattern)
		if err != nil {
			return "", err
		}
		if len(failedReplications) > 0 {
			log.Info(fmt.Sprintf("The replication failed for the following indices: %v", failedReplications))
			return DEGRADED, nil
		} else {
			return UP, nil
		}
	}
	log.Info("Can not recognize replication state")
	return DOWN, nil
}

func (rc ReplicationChecker) listUnhealthyIndices(pattern string) ([]string, error) {
	var indices []string
	responseBody, err := rc.restClient.SendRequestWithStatusCodeCheck(http.MethodGet, catIndicesPath, nil)
	if err != nil {
		log.Error(err, "An error occurred during getting OpenSearch indices")
		return indices, err
	}
	var allIndices []Index
	err = json.Unmarshal(responseBody, &allIndices)
	if err != nil {
		log.Error(err, "An error occurred during unmarshalling OpenSearch indices response")
		return indices, err
	}
	re := regexp.MustCompile(strings.ReplaceAll(pattern, "*", ".*"))
	for _, index := range allIndices {
		if re.MatchString(index.Index) && index.Health != "green" {
			indices = append(indices, index.Index)
		}
	}
	return indices, nil
}

func (rc ReplicationChecker) listFailedReplications(pattern string) ([]string, error) {
	var failedReplications []string
	responseBody, err := rc.restClient.SendRequestWithStatusCodeCheck(http.MethodGet, pattern, nil)
	if err != nil {
		log.Error(err, "An error occurred during getting OpenSearch indices")
		return failedReplications, err
	}
	var indices map[string]interface{}
	err = json.Unmarshal(responseBody, &indices)
	if err != nil {
		log.Error(err, "An error occurred during unmarshalling OpenSearch indices response")
		return failedReplications, err
	}
	for index := range indices {
		if strings.HasPrefix(index, ".") {
			continue
		}
		replicationStatus, err := rc.getIndexReplicationStatus(index)
		if err != nil {
			log.Error(err, fmt.Sprintf("Cannot get replication status of [%s] index", index))
			return failedReplications, err
		}
		if replicationStatus.Status == failedStatus {
			failedReplications = append(failedReplications, index)
		}
	}
	return failedReplications, nil
}

func (rc ReplicationChecker) getIndexReplicationStatus(indexName string) (IndexReplicationStatus, error) {
	var indexReplicationStatus IndexReplicationStatus
	path := fmt.Sprintf(indexReplicationStatusPattern, indexName)
	_, responseBody, err := rc.restClient.SendRequest(http.MethodGet, path, nil)
	if err != nil {
		return indexReplicationStatus, err
	}
	err = json.Unmarshal(responseBody, &indexReplicationStatus)
	return indexReplicationStatus, err
}

func createUrl(host string, port int) string {
	return fmt.Sprintf("http://%s:%d", host, port)
}
