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
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	v1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/strings/slices"

	opensearchservice "github.com/Netcracker/opensearch-service/api/v1"
	"github.com/Netcracker/opensearch-service/util"
	"github.com/go-logr/logr"
)

const (
	opensearchConfigHashName       = "config.opensearch"
	opensearchRoleMappingsHashName = "rolemappings"
	certificateFilePath            = "/certs/crt.pem"
	healthCheckInterval            = 30 * time.Second
	healthCheckTimeout             = 5 * time.Minute
	podCheckInterval               = 1 * time.Minute
	podCheckTimeout                = 6 * time.Minute
	rollingUpdateDoneStatus        = "done"
	rollingUpdateRunningStatus     = "running"
	flushPath                      = "_flush"
	clusterHealthPath              = "_cluster/health"
	clusterSettingsPath            = "_cluster/settings"
	allAccess                      = "all_access"
)

type OpenSearchHealth struct {
	Status string `json:"status"`
}

type FlushResult struct {
	Shards struct {
		Total      int `json:"total"`
		Successful int `json:"successful"`
		Failed     int `json:"failed"`
	} `json:"_shards"`
}

type OpenSearchSettings struct {
	Persistent struct {
		Cluster struct {
			Routing struct {
				Allocation struct {
					Enable *string `json:"enable"`
				} `json:"allocation"`
			} `json:"routing"`
		} `json:"cluster"`
	} `json:"persistent"`
}

type OpenSearchAuditConfig struct {
	Config struct {
		Enabled    bool                   `json:"enabled"`
		Audit      map[string]interface{} `json:"audit"`
		Compliance map[string]interface{} `json:"compliance"`
	} `json:"config"`
}

type OpenSearchReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

type MappingAllAccess struct {
	AllAccess OpenSearchRoleMapping `json:"all_access"`
}
type OpenSearchRoleMapping struct {
	RoleName        string   `json:"role_name,omitempty"`
	Description     string   `json:"description"`
	BackendRoles    []string `json:"backend_roles"`
	AndBackendRoles []string `json:"and_backend_roles"`
	Hosts           []string `json:"hosts"`
	Users           []string `json:"users"`
	Reserved        bool     `json:"reserved,omitempty"`
	Hidden          bool     `json:"hidden,omitempty"`
}

func NewOpenSearchReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) OpenSearchReconciler {
	return OpenSearchReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r OpenSearchReconciler) Reconcile() error {
	if !r.cr.Spec.OpenSearch.RollingUpdate {
		r.logger.Info("Rolling Update is disabled, so skip reconcile procedure")
		return nil
	}

	r.logger.Info("Begin OpenSearch Reconcile procedure.")

	client, err := r.createRestClientWithOldCreds()
	if err != nil {
		r.logger.Error(err, "Error while creating rest client with old creds")
		return err
	}

	statefulSets, err := r.getStatefulSets()
	if err != nil {
		r.logger.Info("Error while searching OpenSearch StatefulSets")
		return err
	}
	if len(statefulSets) == 0 {
		r.logger.Info("Can't process rolling update because there is no stateful set to check," +
			" so finish OpenSearch Reconcile procedure")
		return nil
	}

	perform, err := r.needToPerformRollingUpdate(client, statefulSets)
	if err != nil {
		return err
	}
	if !perform {
		if err := r.updateRollingUpdateStatus(rollingUpdateDoneStatus); err != nil {
			return err
		}
		if err := r.enableAllocationIfNecessary(client); err != nil {
			return err
		}
		r.logger.Info("End OpenSearch Reconcile procedure")
		return nil
	}

	if err = r.runRollingUpdate(client, statefulSets); err != nil {
		return err
	}

	r.logger.Info("End OpenSearch Reconcile procedure")
	return nil
}

func (r OpenSearchReconciler) getStatefulSets() ([]*v1.StatefulSet, error) {
	var statefulSets []*v1.StatefulSet
	if r.cr.Spec.OpenSearch.StatefulSetNames == "" {
		return statefulSets, nil
	}
	for _, statefulSetName := range strings.Split(r.cr.Spec.OpenSearch.StatefulSetNames, ",") {
		statefulSet, err := r.reconciler.watchStatefulSet(statefulSetName, r.cr, r.logger)
		if err != nil {
			return nil, err
		}
		statefulSets = append(statefulSets, statefulSet)
	}
	return statefulSets, nil
}

func (r OpenSearchReconciler) isOpenSearchHealthy(restClient *util.RestClient) (bool, error) {
	r.logger.Info("Check OpenSearch health...")

	responseBody, err := restClient.SendRequestWithStatusCodeCheck(http.MethodGet, clusterHealthPath, nil)
	if err != nil {
		r.logger.Error(err, "Error while getting OpenSearch health status")
		return false, err
	}

	var health OpenSearchHealth
	err = json.Unmarshal(responseBody, &health)
	if err != nil {
		r.logger.Error(err, "Error while unmarshalling OpenSearch health status")
		return false, err
	}

	r.logger.Info(fmt.Sprintf("OpenSearch status: %s", health.Status))
	return health.Status == "green", nil
}

func (r OpenSearchReconciler) checkOpenSearchHealth(restClient *util.RestClient) error {
	r.logger.Info("Checking OpenSearch Health with retries...")

	var lastError error
	err := wait.PollImmediate(healthCheckInterval, healthCheckTimeout, func() (bool, error) {
		var isHealthy bool
		isHealthy, lastError = r.isOpenSearchHealthy(restClient)
		if isHealthy {
			r.logger.Info("OpenSearch is healthy.")
			return true, nil
		}
		r.logger.Info("OpenSearch is not healthy yet ...")
		return false, nil
	})
	if err != nil {
		r.logger.Info(fmt.Sprintf("OpenSearch is not healthy. The last error: %o", lastError))
		return err
	}

	return nil
}

func (r OpenSearchReconciler) needToPerformRollingUpdate(client *util.RestClient, statefulSets []*v1.StatefulSet) (bool, error) {
	for _, statefulSet := range statefulSets {
		if statefulSet.Spec.UpdateStrategy.Type != v1.OnDeleteStatefulSetStrategyType {
			r.logger.Info(fmt.Sprintf("Need to skip Rolling Update, because %s stateful set "+
				"update strategy is not OnDelete", statefulSet.Name))
			return false, nil
		}
	}

	r.logger.Info(fmt.Sprintf("OpenSearch rolling update state in CR: %s", r.cr.Status.RollingUpdateStatus.Status))
	if r.cr.Status.RollingUpdateStatus.Status == rollingUpdateRunningStatus {
		r.logger.Info("Operator Rolling Update state is running, need to continue upgrade")
		return true, nil
	}

	allNodesAlreadyUpdated := true
	for _, statefulSet := range statefulSets {
		if *statefulSet.Spec.Replicas != statefulSet.Status.UpdatedReplicas {
			allNodesAlreadyUpdated = false
			break
		}
	}
	if allNodesAlreadyUpdated {
		r.logger.Info("All OpenSearch nodes are already updated")
		return false, nil
	}

	if err := r.checkOpenSearchHealth(client); err != nil {
		return false, err
	}

	return true, nil
}

func (r OpenSearchReconciler) updateRollingUpdateStatus(status string) error {
	statusUpdater := util.NewStatusUpdater(r.reconciler.Client, r.cr)
	err := statusUpdater.UpdateStatusWithRetry(func(cr *opensearchservice.OpenSearchService) {
		cr.Status.RollingUpdateStatus.Status = status
	})
	if err != nil {
		r.logger.Error(err, fmt.Sprintf("Error while set %s status to CR Rolling Update section", status))
	}
	r.cr.Status.RollingUpdateStatus.Status = status
	return err
}

func (r OpenSearchReconciler) uploadUpdatedReplicasSlice(updatedReplicas []int32, status *opensearchservice.StatefulSetStatus) error {
	r.logger.Info(fmt.Sprintf("Updating replicas with new slise: %o", updatedReplicas))
	status.UpdatedReplicas = updatedReplicas
	return r.updateStatefulSetStatuses()
}

func (r OpenSearchReconciler) updateStatefulSetStatuses() error {
	r.logger.Info("Updating rolling update statuses")
	statusUpdater := util.NewStatusUpdater(r.reconciler.Client, r.cr)
	err := statusUpdater.UpdateStatusWithRetry(func(cr *opensearchservice.OpenSearchService) {
		cr.Status.RollingUpdateStatus.StatefulSetStatuses = r.cr.Status.RollingUpdateStatus.StatefulSetStatuses
	})
	if err != nil {
		r.logger.Error(err, "Error while update stateful set statuses to CR")
	}
	return err
}

func (r OpenSearchReconciler) makeOpenSearchSettings(allocationEnabled bool) OpenSearchSettings {
	settings := OpenSearchSettings{}
	// OpenSearch has "all" value as default for the allocation.enable parameter
	if !allocationEnabled {
		enabled := "primaries"
		settings.Persistent.Cluster.Routing.Allocation.Enable = &enabled
	}
	return settings
}

func (r OpenSearchReconciler) getSettings(client *util.RestClient) (*OpenSearchSettings, error) {
	r.logger.Info("Getting OpenSearch settings...")

	responseBody, err := client.SendRequestWithStatusCodeCheck(http.MethodGet, clusterSettingsPath, nil)
	if err != nil {
		r.logger.Error(err, "Error while getting OpenSearch settings")
		return nil, err
	}

	var returnedSettings OpenSearchSettings
	if err = json.Unmarshal(responseBody, &returnedSettings); err != nil {
		r.logger.Error(err, "Error while unmarshalling settings")
		return nil, err
	}

	return &returnedSettings, nil
}

func (r OpenSearchReconciler) updateSettings(client *util.RestClient, body io.Reader) error {
	r.logger.Info("Updating OpenSearch settings...")

	_, err := client.SendRequestWithStatusCodeCheck(http.MethodPut, clusterSettingsPath, body)
	if err != nil {
		r.logger.Error(err, "Error while updating OpenSearch settings")
		return err
	}

	return nil
}

func (r OpenSearchReconciler) changeAllocationSetting(enableAllocation bool, client *util.RestClient) error {
	if enableAllocation {
		log.Info("Try to enable OpenSearch allocation")
	} else {
		log.Info("Try to disable OpenSearch allocation")
	}

	settings := r.makeOpenSearchSettings(enableAllocation)
	bytes_, err := json.Marshal(settings)
	if err != nil {
		r.logger.Error(err, "Error while marshalling settings with allocation")
		return err
	}
	body := strings.NewReader(string(bytes_))

	r.logger.Info("Try to update settings request with retires...")
	var requestError error
	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		if err = r.updateSettings(client, body); err != nil {
			requestError = err
			return false, nil
		}
		return true, nil
	})
	if err != nil {
		log.Error(err, fmt.Sprintf("Error while sending request. The last error: %o", requestError))
		return err
	}

	returnedSettings, err := r.getSettings(client)
	if err != nil {
		return err
	}

	enabled := returnedSettings.Persistent.Cluster.Routing.Allocation.Enable
	if enabled != nil {
		log.Info(fmt.Sprintf("New allocation status: %s", *enabled))
	} else {
		log.Info("New allocation status: all")
	}

	return nil
}

func (r OpenSearchReconciler) enableAllocationIfNecessary(client *util.RestClient) error {
	r.logger.Info("Checking OpenSearch allocation status...")

	returnedSettings, err := r.getSettings(client)
	if err != nil {
		return err
	}

	if returnedSettings.Persistent.Cluster.Routing.Allocation.Enable == nil ||
		*returnedSettings.Persistent.Cluster.Routing.Allocation.Enable == "all" {
		r.logger.Info("OpenSearch allocation is already enabled")
		return nil
	}

	r.logger.Info("OpenSearch allocation is disabled. Try to enable it")
	return r.changeAllocationSetting(true, client)

}

func (r OpenSearchReconciler) execFlushProcedure(client *util.RestClient) error {
	r.logger.Info("Sending the flush procedure request...")

	responseBody, err := client.SendRequestWithStatusCodeCheck(http.MethodPost, flushPath, nil)
	if err != nil {
		r.logger.Error(err, "Error while requesting flush procedure")
		return err
	}

	var result FlushResult
	if err = json.Unmarshal(responseBody, &result); err != nil {
		r.logger.Error(err, "Error while unmarshalling flush result")
		return err
	}

	if result.Shards.Failed != 0 {
		return fmt.Errorf("flush procedure finished with %d failed shards", result.Shards.Failed)
	}

	sleepPeriod := 20 * time.Second
	r.logger.Info(fmt.Sprintf("Sleep for %s while OpenSearch process flush request ...", sleepPeriod))
	time.Sleep(sleepPeriod)
	return nil
}

func (r OpenSearchReconciler) runRollingUpdate(client *util.RestClient, statefulSets []*v1.StatefulSet) error {
	r.logger.Info("Running Rolling Update procedure...")
	enabledAllocationAfterPodRestart := false
	defer func() {
		if enabledAllocationAfterPodRestart {
			return
		}
		if err := r.changeAllocationSetting(true, client); err != nil {
			r.logger.Error(err, "Error while enabling location after failing in rolling update.")
		}
	}()

	if r.cr.Status.RollingUpdateStatus.Status != rollingUpdateRunningStatus {
		if err := r.changeAllocationSetting(false, client); err != nil {
			return err
		}
		if err := r.execFlushProcedure(client); err != nil {
			return err
		}
		if err := r.updateRollingUpdateStatus(rollingUpdateRunningStatus); err != nil {
			return err
		}
	}

	if err := r.restartOpenSearchPods(statefulSets); err != nil {
		r.logger.Error(err, "Error while OpenSearch pods restarting")
		return err
	}

	enabledAllocationAfterPodRestart = true
	if err := r.changeAllocationSetting(true, client); err != nil {
		r.logger.Error(err, "Error while enabling location after rolling update.")
		return err
	}

	if err := r.checkOpenSearchHealth(client); err != nil {
		return err
	}

	if err := r.updateRollingUpdateStatus(rollingUpdateDoneStatus); err != nil {
		return err
	}

	return nil
}

func (r OpenSearchReconciler) restartOpenSearchPods(statefulSets []*v1.StatefulSet) error {
	for _, statefulSet := range statefulSets {
		status, err := r.findStatefulSetStatus(statefulSet)
		if err != nil {
			return err
		}

		if *statefulSet.Spec.Replicas != statefulSet.Status.UpdatedReplicas {
			if err := r.restartOpenSearchPod(statefulSet, status); err != nil {
				return err
			}
		} else {
			r.logger.Info("All replicas are already updated")
		}

		if len(status.UpdatedReplicas) != 0 {
			r.logger.Info(fmt.Sprintf("Clear %s updated replicas slice in CR", statefulSet.Name))
			status.UpdatedReplicas = []int32{}
			if err := r.updateStatefulSetStatuses(); err != nil {
				return err
			}
		}
	}
	return nil
}

func (r OpenSearchReconciler) findStatefulSetStatus(statefulSet *v1.StatefulSet) (*opensearchservice.StatefulSetStatus, error) {
	r.logger.Info(fmt.Sprintf("Searching rolling update status for %s stateful set", statefulSet.Name))
	statuses := r.cr.Status.RollingUpdateStatus.StatefulSetStatuses
	for index := range statuses {
		status := &(r.cr.Status.RollingUpdateStatus.StatefulSetStatuses[index])
		if status.Name == statefulSet.Name {
			r.logger.Info(fmt.Sprintf("Stateful set status found in CR: status-name: %s, last generation: %d, updated relicas: %o", status.Name, status.LastStatefulSetGeneration, status.UpdatedReplicas))
			return status, nil
		}
	}

	r.logger.Info("Stateful set status was not found. Create new one")
	status := opensearchservice.StatefulSetStatus{Name: statefulSet.Name, LastStatefulSetGeneration: statefulSet.Generation, UpdatedReplicas: []int32{}}
	r.cr.Status.RollingUpdateStatus.StatefulSetStatuses = append(statuses, status)
	if err := r.updateStatefulSetStatuses(); err != nil {
		return nil, err
	}

	lastElementId := len(r.cr.Status.RollingUpdateStatus.StatefulSetStatuses) - 1
	newStatus := &(r.cr.Status.RollingUpdateStatus.StatefulSetStatuses[lastElementId])
	r.logger.Info(fmt.Sprintf("Return new status %s", newStatus.Name))
	return newStatus, nil
}

func (r OpenSearchReconciler) restartOpenSearchPod(statefulSet *v1.StatefulSet, status *opensearchservice.StatefulSetStatus) error {
	updatedReplicas, err := r.getUpdatedReplicasSlice(statefulSet, status)
	if err != nil {
		return err
	}

	for replica := *statefulSet.Spec.Replicas - 1; replica >= 0; replica-- {
		if util.ArrayContains(updatedReplicas, replica) {
			continue
		}

		podName := fmt.Sprintf("%s-%d", statefulSet.Name, replica)
		r.logger.Info(fmt.Sprintf("Try to restart OpenSearch pod %s", podName))
		if err := r.reconciler.deletePodByName(podName, r.cr.Namespace, r.logger); err != nil {
			return err
		}

		if err := r.waitUntilOpenSearchPodIsReady(podName); err != nil {
			return err
		}

		updatedReplicas = append(updatedReplicas, replica)
		if err := r.uploadUpdatedReplicasSlice(updatedReplicas, status); err != nil {
			return err
		}
	}

	return nil
}

func (r OpenSearchReconciler) getUpdatedReplicasSlice(statefulSet *v1.StatefulSet, status *opensearchservice.StatefulSetStatus) ([]int32, error) {
	if statefulSet.Generation != status.LastStatefulSetGeneration {
		r.logger.Info("Current stateful set generation and generation in CR are different, so clear updated replicas slice and update the last generation.")
		status.UpdatedReplicas = []int32{}
		status.LastStatefulSetGeneration = statefulSet.Generation
		return status.UpdatedReplicas, r.updateStatefulSetStatuses()
	}
	return status.UpdatedReplicas, nil
}

func (r OpenSearchReconciler) waitUntilOpenSearchPodIsReady(podName string) error {
	r.logger.Info(fmt.Sprintf("Waiting until %s pod becomes ready...", podName))
	return wait.Poll(podCheckInterval, podCheckTimeout, func() (bool, error) {
		pod, err := r.reconciler.findPod(podName, r.cr.Namespace, r.logger)
		if err != nil {
			return false, nil
		}

		r.logger.Info(fmt.Sprintf("%s pod found, checking container status ...", podName))
		if len(pod.Status.ContainerStatuses) == 0 {
			r.logger.Info(fmt.Sprintf("%s pod doesn't have any container. Skip this check iteration", podName))
			return false, nil
		}

		r.logger.Info(fmt.Sprintf("Container ready: %t", pod.Status.ContainerStatuses[0].Ready))
		return pod.Status.ContainerStatuses[0].Ready, nil
	})
}

func (r OpenSearchReconciler) Status() error {
	return nil
}

func (r OpenSearchReconciler) Configure() error {
	restClient, err := r.processSecurity()
	if err != nil {
		return err
	}

	if r.cr.Spec.OpenSearch.Snapshots != nil {
		if err = r.createSnapshotsRepository(restClient, 5); err != nil {
			return err
		}
	}

	return r.updateCompatibilityMode(restClient)
}

func (r OpenSearchReconciler) processSecurity() (*util.RestClient, error) {
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	client, err := r.reconciler.configureClient()
	if err != nil {
		return nil, err
	}
	newCredentials := r.reconciler.parseSecretCredentials(fmt.Sprintf(secretPattern, r.cr.Name), r.cr.Namespace, r.logger)
	restClient := util.NewRestClient(url, client, newCredentials)
	allaccessRole, err := r.getRoleMapping(restClient, allAccess)
	if err != nil {
		return restClient, err
	}
	if allaccessRole.BackendRoles == nil {
		credentials := r.reconciler.parseSecretCredentials(fmt.Sprintf(secretPattern, r.cr.Name), r.cr.Namespace, r.logger)
		err = r.UpdateRoles(restClient, credentials.Username, allAccess)
		if err != nil {
			return restClient, err
		}
	}
	restClient, err = r.updateCredentials(url, client, newCredentials)
	if err != nil {
		if strings.Contains(err.Error(), "is read-only") {
			clusterManagerPod, requestErr := r.getClusterManagerNode(restClient)
			if requestErr != nil {
				return restClient, requestErr
			}
			commandErr := r.reconciler.runCommandInPod(clusterManagerPod, "opensearch", r.cr.Namespace,
				[]string{"/bin/sh", "-c", "/usr/share/opensearch/bin/reconfiguration.sh"})
			if commandErr != nil {
				return restClient, commandErr
			}
		}
		return restClient, err
	}
	if r.cr.Spec.OpenSearch.DisabledRestCategories != nil {
		err := r.updateAuditConfiguration(restClient)
		if err != nil {
			return restClient, err
		}
	}

	opensearchConfigHash, err :=
		r.reconciler.calculateSecretDataHash(r.cr.Spec.OpenSearch.SecurityConfigurationName, opensearchConfigHashName, r.cr, r.logger)
	if err != nil {
		return restClient, err
	}
	if r.reconciler.ResourceHashes[opensearchConfigHashName] != "" && r.reconciler.ResourceHashes[opensearchConfigHashName] != opensearchConfigHash {
		err := r.updateSecurityConfiguration(restClient)
		if err != nil {
			return restClient, err
		}
	}
	r.reconciler.ResourceHashes[opensearchConfigHashName] = opensearchConfigHash
	opensearchRoleMappingHash, err :=
		r.reconciler.calculateSecretDataHash(fmt.Sprintf("%s-ldap-rolemappings", r.cr.Name), opensearchRoleMappingsHashName, r.cr, r.logger)
	if err == nil {
		if r.reconciler.ResourceHashes[opensearchRoleMappingsHashName] == "" || (r.reconciler.ResourceHashes[opensearchRoleMappingsHashName] != "" && r.reconciler.ResourceHashes[opensearchRoleMappingsHashName] != opensearchRoleMappingHash) {
			err = r.updateLdapRolesmapping(restClient)
			if err != nil {
				return restClient, err
			}
		}
		r.reconciler.ResourceHashes[opensearchRoleMappingsHashName] = opensearchRoleMappingHash
	}
	return restClient, nil
}

func (r OpenSearchReconciler) createRestClientWithOldCreds() (*util.RestClient, error) {
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	client, err := r.reconciler.configureClient()
	if err != nil {
		return nil, err
	}
	oldCredentials := r.reconciler.parseSecretCredentials(fmt.Sprintf(oldSecretPattern, r.cr.Name), r.cr.Namespace, r.logger)
	return util.NewRestClient(url, client, oldCredentials), nil
}

func (r OpenSearchReconciler) updateCredentials(url string, client http.Client, newCredentials util.Credentials) (*util.RestClient, error) {

	oldCredentials := r.reconciler.parseSecretCredentials(fmt.Sprintf(oldSecretPattern, r.cr.Name), r.cr.Namespace, r.logger)
	restClient := util.NewRestClient(url, client, oldCredentials)

	if newCredentials.Username != oldCredentials.Username ||
		newCredentials.Password != oldCredentials.Password {
		if newCredentials.Username != oldCredentials.Username {
			if err := r.createNewUser(newCredentials.Username, newCredentials.Password, restClient); err != nil {
				return restClient, err
			}
			if err := r.removeUser(oldCredentials.Username, restClient); err != nil {
				return restClient, err
			}
		} else {
			if err := r.changeUserPassword(newCredentials.Username, newCredentials.Password, restClient); err != nil {
				return restClient, err
			}
		}
		err := wait.PollImmediate(waitingInterval, updateTimeout, func() (bool, error) {
			err := r.reconciler.updateSecretWithCredentials(fmt.Sprintf(oldSecretPattern, r.cr.Name), r.cr.Namespace, newCredentials, r.logger)
			if err != nil {
				r.logger.Error(err, "Unable to update secret with credentials")
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return restClient, err
		}
		restClient = util.NewRestClient(url, client, newCredentials)
	}
	return restClient, nil
}

func (r OpenSearchReconciler) getClusterManagerNode(restClient *util.RestClient) (string, error) {
	requestPath := "_cat/cluster_manager?h=node"
	statusCode, responseBody, err := restClient.SendBasicRequest(http.MethodGet, requestPath, nil, false)
	if err == nil {
		if statusCode == http.StatusOK {
			return strings.TrimSpace(string(responseBody)), nil
		}
		return "", fmt.Errorf("unable to receive cluster_manager node: [%d] %s", statusCode, responseBody)
	}
	return "", err
}

func (r OpenSearchReconciler) createNewUser(username string, password string, restClient *util.RestClient) error {
	if username == "" || password == "" {
		r.logger.Error(nil, "Unable to create user with empty name or password")
		return nil
	}
	requestPath := fmt.Sprintf("_plugins/_security/api/internalusers/%s", username)
	body := fmt.Sprintf(`{"password": "%s", "description": "Admin user", "backend_roles": ["admin"], 
"opendistro_security_roles": ["all_access", "manage_snapshots"]}`, password)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK || statusCode == http.StatusCreated {
			r.logger.Info("The user is successfully created")
			return nil
		}
		return fmt.Errorf("user creation went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

func (r OpenSearchReconciler) changeUserPassword(username string, password string, restClient *util.RestClient) error {
	if username == "" || password == "" {
		r.logger.Error(nil, "Unable to update user with empty name or password")
		return nil
	}
	requestPath := fmt.Sprintf("_plugins/_security/api/internalusers/%s", username)
	body := fmt.Sprintf(`[{"op": "add", "path": "/password", "value": "%s"}]`, password)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPatch, requestPath, strings.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK {
			r.logger.Info("The password for user is successfully updated")
			return nil
		}
		return fmt.Errorf("user update went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

func (r OpenSearchReconciler) removeUser(username string, restClient *util.RestClient) error {
	if username == "" {
		return nil
	}
	requestPath := fmt.Sprintf("_plugins/_security/api/internalusers/%s", username)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodDelete, requestPath, nil)
	if err == nil {
		if statusCode == http.StatusOK || statusCode == http.StatusNotFound {
			r.logger.Info("The user is successfully deleted")
			return nil
		}
		return fmt.Errorf("user removal went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

// updateAuditConfiguration updates security configuration for audit logging in OpenSearch
func (r OpenSearchReconciler) updateAuditConfiguration(restClient *util.RestClient) error {
	requestPath := "_plugins/_security/api/audit"
	_, responseBody, err := restClient.SendRequest(http.MethodGet, requestPath, nil)
	if err != nil {
		log.Error(err, "An error occurred during audit config request")
		return err
	}
	var auditConfig OpenSearchAuditConfig
	err = json.Unmarshal(responseBody, &auditConfig)
	if err != nil {
		log.Error(err, "An error occurred during unmarshalling audit config")
		return err
	}
	auditConfig.Config.Audit["disabled_rest_categories"] = r.cr.Spec.OpenSearch.DisabledRestCategories
	body, _ := json.Marshal(auditConfig.Config)
	if err != nil {
		log.Error(err, "An error occurred during marshalling audit config")
		return err
	}
	requestPath = "_plugins/_security/api/audit/config"
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPut, requestPath, bytes.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK {
			r.logger.Info("The audit config successfully updated")
			return nil
		}
		return fmt.Errorf("audit config update went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

// updateSecurityConfiguration updates security configuration in OpenSearch
func (r OpenSearchReconciler) updateSecurityConfiguration(restClient *util.RestClient) error {
	secret, err := r.reconciler.findSecret(r.cr.Spec.OpenSearch.SecurityConfigurationName, r.cr.Namespace, r.logger)
	if err != nil {
		return err
	}
	securityConfiguration := secret.Data["config.yml"]
	if securityConfiguration == nil {
		r.logger.Info("Security configuration is empty, so there is nothing to update")
		return nil
	}
	var configuration map[string]interface{}
	if err = yaml.Unmarshal(securityConfiguration, &configuration); err != nil {
		return err
	}
	if configuration["config"] == nil {
		r.logger.Info("Security configuration is empty, so there is nothing to update")
		return nil
	}
	return r.updateSecurityConfig(configuration["config"], restClient)
}

func (r OpenSearchReconciler) updateLdapRolesmapping(restClient *util.RestClient) error {
	newRoleMappings, err := r.getRoleMappingListFromSecret(fmt.Sprintf("%s-ldap-rolemappings", r.cr.Name))
	if err != nil {
		return err
	}
	if newRoleMappings == nil {
		r.logger.Info("Role Mappings list is empty, so there is nothing to update")
		return nil
	}
	oldRoleMappings, err := r.getRoleMappingListFromSecret(fmt.Sprintf("%s-ldap-rolemappings-old", r.cr.Name))
	if err != nil {
		return err
	}
	for _, mapping := range newRoleMappings {
		role := mapping.RoleName
		var oldBackendRoles []string
		for i := range oldRoleMappings {
			if oldRoleMappings[i].RoleName == mapping.RoleName {
				oldBackendRoles = oldRoleMappings[i].BackendRoles
				break
			}
		}
		err = r.updateRoleMappingBackendRoles(role, oldBackendRoles, mapping.BackendRoles, restClient)
		if err != nil {
			return err
		}
	}
	return nil
}

func (r OpenSearchReconciler) getRoleMappingListFromSecret(secretName string) ([]OpenSearchRoleMapping, error) {
	secret, err := r.reconciler.findSecret(secretName, r.cr.Namespace, r.logger)
	if err != nil {
		return nil, err
	}
	secretRoleMappings := secret.Data["rolemappings"]
	var mappings []OpenSearchRoleMapping
	if err = json.Unmarshal(secretRoleMappings, &mappings); err != nil {
		return nil, err
	}
	return mappings, nil
}

func (r OpenSearchReconciler) getRoleMapping(restClient *util.RestClient, roleName string) (OpenSearchRoleMapping, error) {
	requestPath := fmt.Sprintf("_plugins/_security/api/rolesmapping/%s", roleName)
	_, responseBody, err := restClient.SendRequest(http.MethodGet, requestPath, nil)
	if err != nil {
		log.Error(err, "Failed get RoleMapping")
		return OpenSearchRoleMapping{}, err
	}
	var mappings MappingAllAccess
	if err = json.Unmarshal(responseBody, &mappings); err != nil {
		return OpenSearchRoleMapping{}, err
	}
	return mappings.AllAccess, nil
}

func (r OpenSearchReconciler) UpdateRoles(restClient *util.RestClient, userName string, role string) error {
	log.Info(fmt.Sprintf("start Update mapping roles for user:%s ", userName))
	requestPath := "_plugins/_security/api/rolesmapping"
	body := fmt.Sprintf(`[{"op": "add", "path": "/%s", "value": {"backend_roles": ["admin"]}}]`, role)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPatch, requestPath, strings.NewReader(body))
	if err != nil {
		return err
	}

	if statusCode == http.StatusOK {
		requestPathOpendistro := fmt.Sprintf("_plugins/_security/api/internalusers/%s", userName)
		bodyOpendistro := `[{"op": "replace", "path": "/opendistro_security_roles", "value": []}]`
		statusCodeOD, responseBodyOD, err := restClient.SendRequest(http.MethodPatch, requestPathOpendistro, strings.NewReader(bodyOpendistro))
		if err == nil {
			if statusCodeOD == http.StatusOK {
				r.logger.Info("The roles for user are successfully updated")
				return nil
			}
			return fmt.Errorf("opendistro security roles update went wrong: [%d] %s", statusCodeOD, responseBodyOD)
		}
	} else {
		return fmt.Errorf("mapping roles update went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

func (r OpenSearchReconciler) mergeBackendRolesLists(oldSecretList []string, newSecretList []string, listFromOpensearch []string) []string {
	var result []string
	var exclude []string
	for _, role := range oldSecretList {
		if !slices.Contains(newSecretList, role) {
			exclude = append(exclude, role)
		}
	}
	for _, role := range listFromOpensearch {
		if !slices.Contains(exclude, role) {
			result = append(result, role)
		}
	}
	for _, role := range newSecretList {
		if !slices.Contains(result, role) {
			result = append(result, role)
		}
	}
	return result
}

func (r OpenSearchReconciler) updateRoleMappingBackendRoles(role string, oldList []string, newList []string, restClient *util.RestClient) error {
	requestPath := fmt.Sprintf("_plugins/_security/api/rolesmapping/%s", role)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodGet, requestPath, nil)
	if err != nil {
		return err
	}
	if (statusCode != http.StatusOK) && (statusCode != http.StatusNotFound) {
		return fmt.Errorf("can not get rolemapping for role %s: [%d] %s", role, statusCode, responseBody)
	}
	var roleMappingParameters OpenSearchRoleMapping
	if statusCode == http.StatusOK {
		var result map[string]OpenSearchRoleMapping
		if err = json.Unmarshal(responseBody, &result); err != nil {
			r.logger.Error(err, "Error while unmarshalling rolemapping")
			return err
		}
		roleMappingParameters = result[role]
	} else {
		roleMappingParameters.Hosts = []string{}
		roleMappingParameters.Users = []string{}
		roleMappingParameters.AndBackendRoles = []string{}
	}
	finalBackendRoles := r.mergeBackendRolesLists(oldList, newList, roleMappingParameters.BackendRoles)
	roleMappingParameters.Reserved = false
	roleMappingParameters.Hidden = false
	roleMappingParameters.BackendRoles = finalBackendRoles
	bytes_, err := json.Marshal(roleMappingParameters)
	if err != nil {
		r.logger.Error(err, "Error while marshalling rolemapping")
		return err
	}
	statusCode, responseBody, err = restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(string(bytes_)))
	if err == nil {
		if (statusCode == http.StatusOK) || (statusCode == http.StatusCreated) {
			return nil
		}
		return fmt.Errorf("can not update or create rolemapping for role %s: [%d] %s", role, statusCode, responseBody)
	}
	return err
}

func (r OpenSearchReconciler) updateSecurityConfig(configuration interface{}, restClient *util.RestClient) error {
	body, err := json.Marshal(configuration)
	if err != nil {
		return err
	}
	requestPath := "_plugins/_security/api/securityconfig/config"
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPut, requestPath, bytes.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK {
			r.logger.Info("Security configuration is successfully updated")
			return nil
		}
		return fmt.Errorf("security configuration update went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

// createSnapshotsRepository creates snapshots repository in OpenSearch
func (r OpenSearchReconciler) createSnapshotsRepository(restClient *util.RestClient, attemptsNumber int) error {
	r.logger.Info(fmt.Sprintf("Create a snapshot repository with name [%s]", r.cr.Spec.OpenSearch.Snapshots.RepositoryName))
	requestPath := fmt.Sprintf("_snapshot/%s", r.cr.Spec.OpenSearch.Snapshots.RepositoryName)
	requestBody := r.getSnapshotsRepositoryBody()
	var statusCode int
	var err error
	var body []byte
	for i := 0; i < attemptsNumber; i++ {
		statusCode, body, err = restClient.SendRequest(http.MethodGet,
			requestPath, nil)

		// repository recreation is required if snapshots folder is changed not by OpenSearch
		if err == nil && statusCode == http.StatusNotFound && strings.Contains(string(body), "repository_missing_exception") {
			r.deleteSnapshotsRepository(restClient, requestPath)
		}
		statusCode, _, err = restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(requestBody))
		if err == nil && statusCode == http.StatusOK {
			r.logger.Info("Snapshot repository is created")
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("snapshots repository is not created; response status code is %d", statusCode)
}

// deleteSnapshotsRepository deletes snapshots repository in OpenSearch
func (r OpenSearchReconciler) deleteSnapshotsRepository(restClient *util.RestClient, path string) {
	body, err := restClient.SendRequestWithStatusCodeCheck(http.MethodDelete, path, nil)
	if err != nil {
		r.logger.Error(err, fmt.Sprintf("Unable to remove OpenSearch repository: %s", body))
	}
}

func (r OpenSearchReconciler) updateCompatibilityMode(restClient *util.RestClient) error {
	value := "null"
	if r.cr.Spec.OpenSearch.CompatibilityModeEnabled {
		value = "true"
	}
	r.logger.Info(fmt.Sprintf("Update compatibility mode with '%s' value", value))
	requestBody := fmt.Sprintf(`{"persistent": {"compatibility.override_main_response_version": %s}}`, value)
	return r.updateSettings(restClient, strings.NewReader(requestBody))
}

func (r OpenSearchReconciler) getS3Credentials() (string, string) {
	secret, err := r.reconciler.findSecret(r.cr.Spec.OpenSearch.Snapshots.S3.SecretName, r.cr.Namespace, r.logger)
	if err != nil {
		r.logger.Info("Can not find s3-credentials secret, use empty user/password")
		return "", ""
	}
	var keyId []byte
	var keySecret []byte
	keyId = secret.Data["s3-key-id"]
	keySecret = secret.Data["s3-key-secret"]
	return string(keyId), string(keySecret)
}

func (r OpenSearchReconciler) getSnapshotsRepositoryBody() string {
	if r.cr.Spec.OpenSearch.Snapshots.S3 != nil {
		if r.cr.Spec.OpenSearch.Snapshots.S3.GcsEnabled {
			s3Bucket := r.cr.Spec.OpenSearch.Snapshots.S3.Bucket
			return fmt.Sprintf(`{"type": "gcs", "settings": {"bucket": "%s", "client": "default"}}`, s3Bucket)
		}
		if r.cr.Spec.OpenSearch.Snapshots.S3.Enabled {
			s3KeyId, s3KeySecret := r.getS3Credentials()
			s3Bucket := r.cr.Spec.OpenSearch.Snapshots.S3.Bucket
			s3Url := r.cr.Spec.OpenSearch.Snapshots.S3.Url
			s3BasePath := r.cr.Spec.OpenSearch.Snapshots.S3.BasePath
			s3Region := r.cr.Spec.OpenSearch.Snapshots.S3.Region
			s3PathStyleAccess := strconv.FormatBool(r.cr.Spec.OpenSearch.Snapshots.S3.PathStyleAccess)
			return fmt.Sprintf(`{"type": "s3", "settings": {"base_path": "%s", "bucket": "%s", "region": "%s", "endpoint": "%s", "protocol": "http", "access_key": "%s", "secret_key": "%s", "compress": true, "path_style_access": "%s"}}`, s3BasePath, s3Bucket, s3Region, s3Url, s3KeyId, s3KeySecret, s3PathStyleAccess)
		}
	}
	return `{"type": "fs", "settings": {"location": "/usr/share/opensearch/snapshots", "compress": true}}`
}
