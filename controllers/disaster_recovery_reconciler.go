package controllers

import (
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	"strings"
	"time"
)

const (
	drConfigHashName            = "config.disasterRecovery"
	replicationRemoteServiceKey = "remoteCluster"
	replicationPatternKey       = "indicesPattern"
)

type DisasterRecoveryReconciler struct {
	cr                 *opensearchservice.OpenSearchService
	logger             logr.Logger
	reconciler         *OpenSearchServiceReconciler
	replicationWatcher ReplicationWatcher
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
		(r.cr.Status.DisasterRecoveryStatus.Status == "failed")

	drConfigHash, err :=
		r.reconciler.calculateConfigDataHash(r.cr.Spec.DisasterRecovery.ConfigMapName, drConfigHashName, r.cr, r.logger)
	if err != nil {
		return err
	}
	drConfigHashChanged := r.reconciler.ResourceHashes[drConfigHashName] != "" && r.reconciler.ResourceHashes[drConfigHashName] != drConfigHash

	if crCondition || drConfigHashChanged {
		r.replicationWatcher.Lock.Lock()
		checkNeeded := isReplicationCheckNeeded(r.cr)
		if err := r.updateDisasterRecoveryStatus("running",
			"The switchover process for OpenSearch has been started"); err != nil {
			return err
		}

		r.logger.Info("Disable client service")
		if err := r.reconciler.disableClientService(r.cr.Name, r.cr.Namespace, r.logger); err != nil {
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
				err = r.runReplicationProcess(replicationManager)
			}
		}

		if r.cr.Spec.DisasterRecovery.Mode == "active" || r.cr.Spec.DisasterRecovery.Mode == "disable" {
			if checkNeeded {
				var indexNames []string
				indexNames, err = replicationManager.getReplicatedIndices()
				if err != nil {
					log.Error(err, "Can not get replication indices. Replication check is failed.")
				}
				log.Info("Start replication check")
				if err = replicationManager.executeReplicationCheck(indexNames); err != nil {
					log.Error(err, "Replication check is failed.")
				}
			} else {
				comment = "Switchover mode has been changed without replication check"
			}
			if err == nil {
				err = r.stopReplication(replicationManager)
			}
			r.logger.Info("Enable client service")
			if err := r.reconciler.enableClientService(r.cr.Name, r.cr.Namespace, r.logger); err != nil {
				return err
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

func (r DisasterRecoveryReconciler) removePreviousReplication(replicationManager ReplicationManager) error {
	r.logger.Info("Check if autofollow task exists")
	if replicationManager.AutofollowTaskExists() {
		if err := replicationManager.RemoveReplicationRule(); err != nil {
			r.logger.Error(err, "can not delete autofollow replication rule")
			return err
		}
		r.logger.Info("Autofollow task was stopped.")
	} else {
		r.logger.Info("Autofollow task does not exist. ")
	}

	r.logger.Info("Try to stop running replication for indices.")
	if err := replicationManager.StopReplication(); err != nil {
		r.logger.Error(err, "can not stop all running replication tasks")
		return err
	}
	r.logger.Info(fmt.Sprintf("Try to stop running replication for all indices match replication pattern [%s].", replicationManager.pattern))
	if err := replicationManager.StopIndicesByPattern(replicationManager.pattern); err != nil {
		r.logger.Error(err, "can not stop OpenSearch indices by pattern during switchover process to `active` state.")
		return err
	}

	if err := replicationManager.DeleteAdminReplicationTask(); err != nil {
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

func (r DisasterRecoveryReconciler) stopReplication(replicationManager ReplicationManager) error {
	if err := r.removePreviousReplication(replicationManager); err != nil {
		return err
	}
	r.logger.Info("Delete indices by pattern `.tasks`")
	_ = replicationManager.DeleteIndicesByPattern(".tasks")

	r.logger.Info("Replication has been stopped")
	return nil
}

func (r DisasterRecoveryReconciler) getReplicationManager() ReplicationManager {
	cmName := r.cr.Spec.DisasterRecovery.ConfigMapName
	configMap, _ := r.reconciler.findConfigMap(cmName, r.cr.Namespace, r.logger)
	remoteService := configMap.Data[replicationRemoteServiceKey]
	pattern := configMap.Data[replicationPatternKey]
	credentials := r.reconciler.parseSecretCredentials(r.cr, r.logger)
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	restClient := NewRestClient(url, r.reconciler.createHttpClient(), credentials)
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
