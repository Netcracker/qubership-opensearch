package controllers

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

const (
	leaderAlias              = "leader-cluster"
	startFullReplicationPath = "_plugins/_replication/_autofollow"
	replicationName          = "dr-replication"
)

//TODO: check this class
type ReplicationManager struct {
	restClient RestClient
	remoteUrl  string
	pattern    string
}

func NewReplicationManager(restClient RestClient, remoteUrl string, indexPattern string) *ReplicationManager {
	return &ReplicationManager{
		restClient: restClient,
		remoteUrl:  remoteUrl,
		pattern:    indexPattern,
	}
}

func (rm ReplicationManager) Configure() error {
	path := "_cluster/settings"
	body := fmt.Sprintf(`{"persistent": {"cluster": {"remote": {"%s": {"seeds": [ "%s" ]}}}}}`, leaderAlias, rm.remoteUrl)
	//TODO: should we process requestBody?
	statusCode, _, err := rm.restClient.sendRequest(http.MethodPut, path, strings.NewReader(body))
	if err != nil {
		return err
	}
	if statusCode >= 500 {
		return errors.New("internal server error")
	}
	return nil
}

func (rm ReplicationManager) Start() error {
	body := fmt.Sprintf(`{"leader_alias":"%s","pattern":"%s","name":"%s","use_roles":{"leader_cluster_role":"all_access","follower_cluster_role":"all_access"}}`,
		leaderAlias, rm.pattern, replicationName)
	statusCode, _, err := rm.restClient.sendRequest(http.MethodPost, startFullReplicationPath, strings.NewReader(body))
	if err != nil {
		return err
	}
	if statusCode >= 500 {
		return errors.New("internal server error")
	}
	return nil
}

func (rm ReplicationManager) Stop() error {
	body := fmt.Sprintf(`{"leader_alias": "%s","name": "%s"}`, leaderAlias, replicationName)
	statusCode, _, err := rm.restClient.sendRequest(http.MethodDelete, startFullReplicationPath, strings.NewReader(body))
	if err != nil {
		return err
	}
	if statusCode >= 500 {
		return errors.New("internal server error")
	}
	return nil
}
