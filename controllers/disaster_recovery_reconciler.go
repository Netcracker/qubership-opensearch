package controllers

import (
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	"net/http"
	"strings"
	"time"
)

const (
	drConfigHashName            = "config.disasterRecovery"
	replicationRemoteServiceKey = "remoteCluster"
	replicationPatternKey       = "indicesPattern"
)

type DisasterRecoveryReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewDisasterRecoveryReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) DisasterRecoveryReconciler {
	return DisasterRecoveryReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
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
		checkNeeded := isReplicationCheckNeeded(r.cr)
		if err := r.updateDisasterRecoveryStatus("running",
			"The switchover process for OpenSearch has been started"); err != nil {
			return err
		}

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
				indexNames, err := replicationManager.getReplicatedIndices()
				if err != nil {
					log.Error(err, "Can not get replication indices. Replication check is failed.")
				}
				if err = replicationManager.executeReplicationCheck(indexNames); err != nil {
					log.Error(err, "Replication check is failed.")
				}
			} else {
				comment = "Switchover mode has been changed without replication check"
			}
			if err == nil {
				err = r.stopReplication(replicationManager)
			}
		}

		defer func() {
			if status == "failed" {
				_ = r.updateDisasterRecoveryStatus(status, comment)
			} else {
				if err != nil {
					status = "failed"
					comment = fmt.Sprintf("Error occurred during OpenSearch switching: %v", err)
				}
				_ = r.updateDisasterRecoveryStatus(status, comment)
			}
			r.logger.Info("Disaster recovery status was updated.")
		}()
	}

	r.reconciler.ResourceHashes[drConfigHashName] = drConfigHash

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
	if !replicationManager.AutofollowTaskExists() {
		r.logger.Info("Autofollower task does not exist")
		return nil
	}
	if err := replicationManager.RemoveReplicationRule(); err != nil {
		r.logger.Error(err, "can not delete autofollow replication rule")
		return err
	}
	if err := replicationManager.StopReplication(); err != nil {
		r.logger.Error(err, "can not stop all running replication tasks")
		return err
	}
	replicationManager.DeleteAdminReplicationTask()
	return nil
}

func (r DisasterRecoveryReconciler) runReplicationProcess(replicationManager ReplicationManager) error {
	if err := replicationManager.DeleteIndices(); err != nil {
		r.logger.Error(err, "can not delete OpenSearch indices by pattern during switchover process to `standby` state.")
		return err
	}
	time.Sleep(time.Second * 2)
	if err := replicationManager.Configure(); err != nil {
		r.logger.Error(err, "can not configure replication connection between DR OpenSearch clusters.")
		return err
	}
	if err := replicationManager.Start(); err != nil {
		r.logger.Error(err, "can not create autofollow replication rule")
		return err
	}
	r.logger.Info("Replication has been started")
	return nil
}

func (r DisasterRecoveryReconciler) stopReplication(replicationManager ReplicationManager) error {
	if !replicationManager.AutofollowTaskExists() {
		r.logger.Info("Autofollow task does not exist. Replication was stopped.")
		return nil
	}
	if err := replicationManager.RemoveReplicationRule(); err != nil {
		r.logger.Error(err, "can not delete autofollow replication rule")
		return err
	}
	if err := replicationManager.StopReplication(); err != nil {
		r.logger.Error(err, "can not stop all running replication tasks")
		return err
	}
	_ = replicationManager.DeleteIndicesByPattern(".tasks")

	if err := replicationManager.StopIndicesByPattern(replicationManager.pattern); err != nil {
		r.logger.Error(err, "can not stop OpenSearch indices by pattern during switchover process to `active` state.")
		return err
	}

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
	restClient := NewRestClient(url, http.Client{}, credentials)
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
