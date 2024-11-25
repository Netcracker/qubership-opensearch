package controllers

import (
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
)

const (
	monitoringSecretHashName = "secret.monitoring"
	monitoringSpecHashName   = "spec.monitoring"
)

type MonitoringReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewMonitoringReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) MonitoringReconciler {
	return MonitoringReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r MonitoringReconciler) Reconcile() error {
	var monitoringSecretHash string
	if r.cr.Spec.Monitoring.SecretName != "" {
		var err error
		monitoringSecretHash, err =
			r.reconciler.calculateSecretDataHash(r.cr.Spec.Monitoring.SecretName, monitoringSecretHashName, r.cr, r.logger)
		if err != nil {
			return err
		}
	}

	opensearchOldSecretHash, err :=
		r.reconciler.calculateSecretDataHash(fmt.Sprintf(oldSecretPattern, r.cr.Name), opensearchOldSecretHashName, r.cr, r.logger)
	if err != nil {
		return err
	}

	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[opensearchOldSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchOldSecretHashName] != opensearchOldSecretHash ||
		r.reconciler.ResourceHashes[monitoringSecretHashName] != "" && r.reconciler.ResourceHashes[monitoringSecretHashName] != monitoringSecretHash {
		annotations := map[string]string{
			opensearchSecretHashName:    opensearchSecretHash,
			opensearchOldSecretHashName: opensearchOldSecretHash,
			monitoringSecretHashName:    monitoringSecretHash,
		}

		if err := r.reconciler.addAnnotationsToDeployment(r.cr.Spec.Monitoring.Name, r.cr.Namespace, annotations, r.logger); err != nil {
			return err
		}
	}

	monitoringSpecHash, err := util.Hash(r.cr.Spec.Monitoring)
	if err != nil {
		return err
	}
	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[opensearchOldSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchOldSecretHashName] != opensearchOldSecretHash ||
		r.reconciler.ResourceHashes[monitoringSpecHashName] != monitoringSpecHash {
		if r.cr.Spec.Monitoring.SlowQueries != nil || *r.reconciler.SlowLogIndicesWatcher.State != stoppedWatcherState {
			helper := r.prepareSlowLogIndicesHelper()
			if r.cr.Spec.Monitoring.SlowQueries != nil {
				r.reconciler.SlowLogIndicesWatcher.start(helper, r.cr.Spec.Monitoring.SlowQueries.IndicesPattern,
					r.cr.Spec.Monitoring.SlowQueries.MinSeconds)
			} else {
				r.reconciler.SlowLogIndicesWatcher.stop(helper)
			}
		}
	}

	r.reconciler.ResourceHashes[opensearchOldSecretHashName] = opensearchOldSecretHash
	r.reconciler.ResourceHashes[monitoringSecretHashName] = monitoringSecretHash
	r.reconciler.ResourceHashes[monitoringSpecHashName] = monitoringSpecHash
	return nil
}

func (r MonitoringReconciler) Status() error {
	return nil
}

func (r MonitoringReconciler) Configure() error {
	return nil
}

func (r MonitoringReconciler) prepareSlowLogIndicesHelper() SlowLogIndicesHelper {
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	client, _ := r.reconciler.configureClient()
	credentials := r.reconciler.parseOpenSearchCredentials(r.cr, r.logger)
	r.logger.Info(fmt.Sprintf("cred = %s", credentials.Password))
	return SlowLogIndicesHelper{
		logger:     r.logger,
		restClient: util.NewRestClient(url, client, credentials),
	}
}
