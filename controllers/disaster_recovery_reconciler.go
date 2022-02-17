package controllers

import (
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	"net/http"
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
		(r.cr.Status.DisasterRecoveryStatus.Status == "failed" &&
			r.cr.Status.DisasterRecoveryStatus.Comment == util.RetryFailedComment)

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
		if checkNeeded {
			// add some logic here for replication check
		} else {
			if r.cr.Spec.DisasterRecovery.Mode == "standby" {
				err = r.runReplicationProcess()
			}
			if r.cr.Spec.DisasterRecovery.Mode == "active" &&
				r.cr.Status.DisasterRecoveryStatus.Mode == "standby" {
				err = r.stopReplication()
			}
			comment = "Switchover mode has been changed without replication check"
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
		}()
	}

	r.reconciler.ResourceHashes[drConfigHashName] = drConfigHash

	return nil
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

func (r DisasterRecoveryReconciler) runReplicationProcess() error {
	replicationManager := r.getReplicationManager()
	if err := replicationManager.Configure(); err != nil {
		return err
	}
	if err := replicationManager.Start(); err != nil {
		return err
	}
	r.logger.Info("Replication has been started")
	return nil
}

func (r DisasterRecoveryReconciler) stopReplication() error {
	replicationManager := r.getReplicationManager()
	if err := replicationManager.Stop(); err != nil {
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
	return *NewReplicationManager(*restClient, remoteService, pattern)
}

func isReplicationCheckNeeded(instance *opensearchservice.OpenSearchService) bool {
	return false
}
