package controllers

import (
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
)

const (
	monitoringSecretHashName = "secret.monitoring"
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

	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[monitoringSecretHashName] != "" && r.reconciler.ResourceHashes[monitoringSecretHashName] != monitoringSecretHash {
		annotations := map[string]string{
			opensearchSecretHashName: opensearchSecretHash,
			monitoringSecretHashName: monitoringSecretHash,
		}

		if err := r.reconciler.addAnnotationsToDeployment(r.cr.Spec.Monitoring.Name, r.cr.Namespace, annotations, r.logger); err != nil {
			return err
		}
	}

	r.reconciler.ResourceHashes[monitoringSecretHashName] = monitoringSecretHash
	return nil
}

func (r MonitoringReconciler) Status() error {
	return nil
}

func (r MonitoringReconciler) Configure() error {
	return nil
}
