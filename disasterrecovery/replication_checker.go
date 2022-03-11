package disasterrecovery

import (
	"encoding/json"
	"fmt"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/controllers"
	"net/http"
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
				if rule.FailedStart == 0 {
					return UP, nil
				}
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
	}
	log.Info("Can not recognize replication state")
	return DOWN, nil
}

func createUrl(host string, port int) string {
	return fmt.Sprintf("http://%s:%d", host, port)
}
