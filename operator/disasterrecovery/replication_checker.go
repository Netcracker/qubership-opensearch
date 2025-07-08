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

package disasterrecovery

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/Netcracker/opensearch-service/operator/util"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"
)

const (
	certificateFilePath           = "/certs/crt.pem"
	catIndicesPath                = "_cat/indices?h=index,health&format=json"
	indexReplicationStatusPattern = "_plugins/_replication/%s/_status"
	failedStatus                  = "FAILED"
	opensearchHostEnvVar          = "OPENSEARCH_HOST"
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

func NewReplicationChecker(opensearchName string, opensearchProtocol string, username string, password string) ReplicationChecker {
	var credentials util.Credentials
	if username != "" && password != "" {
		credentials = util.NewCredentials(username, password)
	}
	url := createUrl(opensearchProtocol, opensearchName, 9200)
	restClient := util.NewRestClient(url, configureClient(), credentials)
	return ReplicationChecker{
		restClient: *restClient,
	}
}

func NewReplicationCheckerWithClient(restClient util.RestClient) ReplicationChecker {
	return ReplicationChecker{
		restClient: restClient,
	}
}

type ReplicationChecker struct {
	restClient util.RestClient
}

func (rc ReplicationChecker) CheckReplication() (string, error) {
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
			failedIndices := util.FilterSlice(rule.FailedIndices, func(s string) bool {
				return !strings.HasPrefix(s, ".")
			})
			if len(failedIndices) == 0 {
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
		failedReplicationsFound, err := rc.areFailedReplicationsFound(rule.Pattern)
		if err != nil {
			return "", err
		}
		if failedReplicationsFound {
			log.Info("The replication failed for some indices")
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
		if re.MatchString(index.Index) && index.Health == "red" {
			indices = append(indices, index.Index)
		}
	}
	return indices, nil
}

func (rc ReplicationChecker) areFailedReplicationsFound(pattern string) (bool, error) {
	responseBody, err := rc.restClient.SendRequestWithStatusCodeCheck(http.MethodGet, pattern, nil)
	if err != nil {
		log.Error(err, "An error occurred during getting OpenSearch indices")
		return true, err
	}
	var indices map[string]interface{}
	err = json.Unmarshal(responseBody, &indices)
	if err != nil {
		log.Error(err, "An error occurred during unmarshalling OpenSearch indices response")
		return true, err
	}
	for index := range indices {
		if strings.HasPrefix(index, ".") {
			continue
		}
		replicationStatus, err := rc.getIndexReplicationStatus(index)
		if err != nil {
			log.Error(err, fmt.Sprintf("Cannot get replication status of [%s] index", index))
			return true, err
		}
		if replicationStatus.Status == failedStatus {
			log.Error(err, fmt.Sprintf("Replication of [%s] index failed", index))
			return true, nil
		}
	}
	return false, nil
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

func configureClient() http.Client {
	httpClient := http.Client{Timeout: time.Second * 5}
	if _, err := os.Stat(certificateFilePath); errors.Is(err, os.ErrNotExist) {
		return httpClient
	}
	caCert, _ := os.ReadFile(certificateFilePath)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	return httpClient
}

func createUrl(protocol string, host string, port int) string {
	osHost := os.Getenv(opensearchHostEnvVar)
	if osHost != "" {
		return osHost
	}
	return fmt.Sprintf("%s://%s-internal:%d", protocol, host, port)
}
