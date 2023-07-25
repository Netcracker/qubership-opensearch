package controllers

import (
	"encoding/json"
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/disasterrecovery"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"strings"
	"time"
)

const (
	drConfigHashName            = "config.disasterRecovery"
	leaderStatsPath             = "_plugins/_replication/leader_stats"
	replicationRemoteServiceKey = "remoteCluster"
	replicationPatternKey       = "indicesPattern"
	interval                    = 10 * time.Second
	timeout                     = 240 * time.Second
	usersRecoveryDoneState      = "done"
	usersRecoveryFailedState    = "failed"
	usersRecoveryIdleState      = "idle"
	usersRecoveryRunningState   = "running"
)

type DisasterRecoveryReconciler struct {
	cr                 *opensearchservice.OpenSearchService
	logger             logr.Logger
	reconciler         *OpenSearchServiceReconciler
	replicationWatcher ReplicationWatcher
}

type LeaderStats struct {
	NumReplicatedIndices        int                    `json:"num_replicated_indices"`
	OperationsRead              int                    `json:"operations_read"`
	TranslogSizeBytes           int                    `json:"translog_size_bytes"`
	OperationsReadLucene        int                    `json:"operations_read_lucene"`
	OperationsReadTranslog      int                    `json:"operations_read_translog"`
	TotalReadTimeLuceneMillis   int                    `json:"total_read_time_lucene_millis"`
	TotalReadTimeTranslogMillis int                    `json:"total_read_time_translog_millis"`
	BytesRead                   int                    `json:"bytes_read"`
	IndexStats                  map[string]interface{} `json:"index_stats"`
}

func NewDisasterRecoveryReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) DisasterRecoveryReconciler {
	return DisasterRecoveryReconciler{
		cr:                 cr,
		logger:             logger,
		reconciler:         r,
		replicationWatcher: r.ReplicationWatcher,
	}
}

func (r DisasterRecoveryReconciler) Reconcile() error {
	return nil
}

func (r DisasterRecoveryReconciler) Status() error {
	return nil
}

func (r DisasterRecoveryReconciler) Configure() error {
	crCondition := r.cr.Spec.DisasterRecovery.Mode != r.cr.Status.DisasterRecoveryStatus.Mode ||
		r.cr.Status.DisasterRecoveryStatus.Status == "running" ||
		r.cr.Status.DisasterRecoveryStatus.Status == "failed"

	drConfigHash, err :=
		r.reconciler.calculateConfigDataHash(r.cr.Spec.DisasterRecovery.ConfigMapName, drConfigHashName, r.cr, r.logger)
	if err != nil {
		return err
	}
	drConfigHashChanged := r.reconciler.ResourceHashes[drConfigHashName] != "" && r.reconciler.ResourceHashes[drConfigHashName] != drConfigHash

	if crCondition || drConfigHashChanged {
		r.replicationWatcher.pause(r.logger)
		r.replicationWatcher.Lock.Lock()
		checkNeeded := isReplicationCheckNeeded(r.cr)
		if err = r.updateDisasterRecoveryStatus("running",
			"The switchover process for OpenSearch has been started"); err != nil {
			return err
		}

		r.logger.Info("Disable client service")
		if err = r.reconciler.disableClientService(r.cr.Name, r.cr.Namespace, r.logger); err != nil {
			return err
		}
		time.Sleep(time.Second * 2)

		status := "done"
		comment := "replication has finished successfully"

		replicationManager := r.getReplicationManager()
		if r.cr.Spec.DisasterRecovery.Mode == "standby" {
			if r.cr.Status.DisasterRecoveryStatus.Mode != "active" {
				r.logger.Info("Removing previous replication rule")
				err = r.removePreviousReplication(replicationManager)
			}
			if err == nil {
				r.logger.Info("Checking existence of active replications")
				err = r.checkExistingReplications(replicationManager)
			}
			if err == nil {
				err = r.runReplicationProcess(replicationManager)
			}
			if err == nil {
				err = r.checkReplication(replicationManager.restClient)
			}
		}

		if r.cr.Spec.DisasterRecovery.Mode == "active" || r.cr.Spec.DisasterRecovery.Mode == "disable" {
			err = r.replicationWatcher.checkReplication(r, true, r.logger)
			if err == nil && checkNeeded {
				var indexNames []string
				indexNames, err = replicationManager.getReplicatedIndices()
				if err != nil {
					r.logger.Error(err, "Can not get replication indices. Replication check is failed.")
				}
				r.logger.Info("Start replication check")
				if err = replicationManager.executeReplicationCheck(indexNames); err != nil {
					r.logger.Error(err, "Replication check is failed.")
				}
			} else {
				comment = "Switchover mode has been changed without replication check"
			}
			if err == nil {
				err = r.stopReplication(replicationManager)
			}
			if err == nil {
				r.logger.Info("Enable client service")
				err = r.reconciler.enableClientService(r.cr.Name, r.cr.Namespace, r.logger)
			}
			if err == nil && crCondition && r.cr.Spec.DbaasAdapter != nil {
				err = r.reconciler.scaleDeploymentForNoWait(r.cr.Spec.DbaasAdapter.Name, r.cr.Namespace, 1, false, r.logger)
				if err == nil {
					r.logger.Info("Start users recovery")
					err = r.recoverUsers()
				}
			}
		}

		defer func() {
			r.replicationWatcher.Lock.Unlock()
			if status == "failed" {
				_ = r.updateDisasterRecoveryStatus(status, comment)
			} else {
				if err != nil {
					status = "failed"
					comment = fmt.Sprintf("Error occurred during OpenSearch switching: %v", err)
				}
				_ = r.updateDisasterRecoveryStatus(status, comment)
			}
			if r.cr.Spec.DisasterRecovery.Mode == "active" {
				_ = r.reconciler.enableClientService(r.cr.Name, r.cr.Namespace, r.logger)
			}
			r.logger.Info("Disaster recovery status was updated.")
		}()
	}

	r.reconciler.ResourceHashes[drConfigHashName] = drConfigHash

	if r.cr.Spec.DisasterRecovery.ReplicationWatcherEnabled {
		r.replicationWatcher.start(r, r.logger)
	} else {
		r.replicationWatcher.pause(r.logger)
	}
	return err
}

// updateDisasterRecoveryStatus updates state of Disaster Recovery switchover
func (r DisasterRecoveryReconciler) updateDisasterRecoveryStatus(status string, comment string) error {
	statusUpdater := util.NewStatusUpdater(r.reconciler.Client, r.cr)
	return statusUpdater.UpdateStatusWithRetry(func(instance *opensearchservice.OpenSearchService) {
		instance.Status.DisasterRecoveryStatus.Mode = r.cr.Spec.DisasterRecovery.Mode
		instance.Status.DisasterRecoveryStatus.Status = status
		instance.Status.DisasterRecoveryStatus.Comment = comment
	})
}

func (r DisasterRecoveryReconciler) updateUsersRecoveryStatus(state string) error {
	statusUpdater := util.NewStatusUpdater(r.reconciler.Client, r.cr)
	return statusUpdater.UpdateStatusWithRetry(func(cr *opensearchservice.OpenSearchService) {
		cr.Status.DisasterRecoveryStatus.UsersRecoveryState = state
	})
}

func (r DisasterRecoveryReconciler) removePreviousReplication(replicationManager ReplicationManager) error {
	if err := replicationManager.RemoveReplicationRule(); err != nil {
		r.logger.Error(err, "can not delete autofollow replication rule")
		return err
	}
	r.logger.Info("Autofollow task was stopped.")

	r.logger.Info("Try to stop running replication for indices.")
	if err := replicationManager.StopReplication(); err != nil {
		r.logger.Error(err, "can not stop all running replication tasks")
		return err
	}
	r.logger.Info(fmt.Sprintf("Try to stop running replication for all indices match replication pattern [%s].", replicationManager.pattern))
	if err := replicationManager.StopIndicesReplicationByPattern(replicationManager.pattern); err != nil {
		r.logger.Error(err, "can not stop OpenSearch indices by pattern during switchover process to `active` state.")
		return err
	}

	if err := replicationManager.DeleteAdminReplicationTasks(); err != nil {
		r.logger.Error(err, "can not delete replication tasks during switchover process to `active` state.")
		return err
	}

	r.logger.Info("Replication has been stopped")
	return nil
}

func (r DisasterRecoveryReconciler) runReplicationProcess(replicationManager ReplicationManager) error {
	r.logger.Info("Delete replication indices")
	if err := replicationManager.DeleteIndices(); err != nil {
		r.logger.Error(err, "can not delete OpenSearch indices by pattern during switchover process to `standby` state.")
		return err
	}
	time.Sleep(time.Second * 2)
	r.logger.Info("Configure replication connection between clusters")
	if err := replicationManager.Configure(); err != nil {
		r.logger.Error(err, "can not configure replication connection between DR OpenSearch clusters.")
		return err
	}
	r.logger.Info("Start autofollow replication")
	if err := replicationManager.Start(); err != nil {
		r.logger.Error(err, "can not create autofollow replication rule")
		return err
	}
	r.logger.Info("Replication has been started")
	return nil
}

func (r DisasterRecoveryReconciler) checkReplication(restClient util.RestClient) error {
	replicationChecker := disasterrecovery.NewReplicationCheckerWithClient(restClient)
	err := wait.Poll(interval, timeout, func() (bool, error) {
		status, err := replicationChecker.CheckReplication()
		if err != nil {
			r.logger.Error(err, "Unable to get replication state")
			return false, nil
		}
		if status != disasterrecovery.UP {
			r.logger.Info("Replication is not healthy yet")
			return false, nil
		}
		r.logger.Info("Replication is healthy")
		return true, nil
	})
	return err
}

func (r DisasterRecoveryReconciler) stopReplication(replicationManager ReplicationManager) error {
	if err := r.removePreviousReplication(replicationManager); err != nil {
		return err
	}
	r.logger.Info("Delete indices by pattern `.tasks`")
	_ = replicationManager.DeleteIndicesByPattern(".tasks")

	r.logger.Info("Replication has been stopped")
	return nil
}

func (r DisasterRecoveryReconciler) recoverUsers() error {
	aggregatorRestClient := r.buildAggregatorRestClient()
	adapterRestClient := r.buildAdapterRestClient()
	data := fmt.Sprintf(`{
		"physicalDbId": "%s",
		"type": "opensearch",
		"settings": {}
	}`, r.cr.Spec.DbaasAdapter.PhysicalDatabaseIdentifier)

	state := r.cr.Status.DisasterRecoveryStatus.UsersRecoveryState
	if state == "" {
		err := r.updateUsersRecoveryStatus(usersRecoveryDoneState)
		r.logger.Info("Users recovery is not run during installation")
		return err
	}
	if state != usersRecoveryRunningState {
		state = usersRecoveryIdleState
	}
	for state != usersRecoveryDoneState && state != usersRecoveryFailedState {
		if state == usersRecoveryIdleState {
			err := wait.PollImmediate(interval, timeout, func() (bool, error) {
				statusCode, response, err := aggregatorRestClient.SendRequest(http.MethodPost,
					"api/v3/dbaas/internal/physical_databases/users/restore-password", strings.NewReader(data))
				if err != nil || statusCode != http.StatusOK {
					r.logger.Error(err, fmt.Sprintf("Unable to restore user passwords via DBaaS aggregator: [%d] %s",
						statusCode, string(response)))
					return false, nil
				}
				return true, nil
			})
			if err != nil {
				state = usersRecoveryFailedState
				continue
			}
			if err = r.updateUsersRecoveryStatus(usersRecoveryRunningState); err != nil {
				return err
			}
		}
		time.Sleep(time.Second * 5)
		statusCode, response, err := adapterRestClient.SendRequest(http.MethodGet,
			"api/v2/dbaas/adapter/opensearch/users/restore-password/state", nil)
		if err != nil || statusCode != http.StatusOK {
			r.logger.Error(err, fmt.Sprintf("Unable to get state of procedure: %s", string(response)))
			continue
		}
		state = string(response)
	}
	err := r.updateUsersRecoveryStatus(state)
	r.logger.Info(fmt.Sprintf("Users recovery is finished with [%s] state", state))
	if err == nil && state == usersRecoveryFailedState {
		return fmt.Errorf("unable to restore OpenSearch users during switchover")
	}
	return err
}

func (r DisasterRecoveryReconciler) buildAggregatorRestClient() *util.RestClient {
	client, _ := r.reconciler.configureClientWithCertificate(dbaasCertificateFilePath)
	credentials := r.reconciler.parseSecretCredentialsByKeys(r.cr.Spec.DbaasAdapter.SecretName, r.cr.Namespace,
		"registration-auth-username", "registration-auth-password", r.logger)
	return util.NewRestClient(r.cr.Spec.DbaasAdapter.AggregatorAddress, client, credentials)
}

func (r DisasterRecoveryReconciler) buildAdapterRestClient() *util.RestClient {
	client, _ := r.reconciler.configureClientWithCertificate(dbaasCertificateFilePath)
	credentials := r.reconciler.parseSecretCredentials(r.cr.Spec.DbaasAdapter.SecretName, r.cr.Namespace, r.logger)
	return util.NewRestClient(r.cr.Spec.DbaasAdapter.AdapterAddress, client, credentials)
}

func (r DisasterRecoveryReconciler) checkExistingReplications(replicationManager ReplicationManager) error {
	responseBody, err := replicationManager.restClient.SendRequestWithStatusCodeCheck(http.MethodGet, leaderStatsPath, nil)
	if err != nil {
		r.logger.Error(err, "An error occurred during getting OpenSearch leader stats")
		return err
	}
	var leaderStats LeaderStats
	err = json.Unmarshal(responseBody, &leaderStats)
	if err != nil {
		r.logger.Error(err, "An error occurred during unmarshalling OpenSearch leader stats response")
		return err
	}
	if len(leaderStats.IndexStats) > 0 {
		r.logger.Error(err, "There is active replication on the other side. To move current side into standby mode, need to move opposite side to active mode first.")
		return fmt.Errorf("there is active replication on the other side")
	}

	r.logger.Info("There are no replications from the other side")
	return nil
}

func (r DisasterRecoveryReconciler) getReplicationManager() ReplicationManager {
	cmName := r.cr.Spec.DisasterRecovery.ConfigMapName
	configMap, _ := r.reconciler.findConfigMap(cmName, r.cr.Namespace, r.logger)
	remoteService := configMap.Data[replicationRemoteServiceKey]
	pattern := configMap.Data[replicationPatternKey]
	credentials := r.reconciler.parseOpenSearchCredentials(r.cr, r.logger)
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	client, _ := r.reconciler.configureClient()
	restClient := util.NewRestClient(url, client, credentials)
	return *NewReplicationManager(*restClient, remoteService, pattern, r.logger)
}

func isReplicationCheckNeeded(instance *opensearchservice.OpenSearchService) bool {
	if instance.Spec.DisasterRecovery.NoWait {
		return false
	}
	specMode := strings.ToLower(instance.Spec.DisasterRecovery.Mode)
	statusMode := strings.ToLower(instance.Status.DisasterRecoveryStatus.Mode)
	switchoverStatus := strings.ToLower(instance.Status.DisasterRecoveryStatus.Status)
	return specMode == "active" && (statusMode != "active" || statusMode == "active" && switchoverStatus == "failed")
}
