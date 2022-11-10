package controllers

import (
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
	"strings"
)

const (
	elasticsearchDbaasAdapterSecretHashName = "secret.elasticsearchDbaasAdapter"
)

type ElasticsearchDbaasAdapterReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewElasticsearchDbaasAdapterReconciler(r *OpenSearchServiceReconciler,
	cr *opensearchservice.OpenSearchService, logger logr.Logger) ElasticsearchDbaasAdapterReconciler {
	return ElasticsearchDbaasAdapterReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r ElasticsearchDbaasAdapterReconciler) Reconcile() error {
	elasticsearchDbaasAdapterSecretHash, err :=
		r.reconciler.calculateSecretDataHash(r.cr.Spec.ElasticsearchDbaasAdapter.SecretName,
			elasticsearchDbaasAdapterSecretHashName, r.cr, r.logger)
	if err != nil {
		return err
	}

	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[elasticsearchDbaasAdapterSecretHashName] != "" && r.reconciler.ResourceHashes[elasticsearchDbaasAdapterSecretHashName] != elasticsearchDbaasAdapterSecretHash {
		annotations := map[string]string{
			opensearchSecretHashName:                opensearchSecretHash,
			elasticsearchDbaasAdapterSecretHashName: elasticsearchDbaasAdapterSecretHash,
		}

		if err := r.reconciler.addAnnotationsToDeployment(r.cr.Spec.ElasticsearchDbaasAdapter.Name, r.cr.Namespace, annotations, r.logger); err != nil {
			return err
		}
	}

	r.reconciler.ResourceHashes[elasticsearchDbaasAdapterSecretHashName] = elasticsearchDbaasAdapterSecretHash
	return nil
}

func (r ElasticsearchDbaasAdapterReconciler) Status() error {
	return nil
}

func (r ElasticsearchDbaasAdapterReconciler) Configure() error {
	if r.cr.Status.DisasterRecoveryStatus.Mode != r.cr.Spec.DisasterRecovery.Mode || r.cr.Status.DisasterRecoveryStatus.Status == "failed" {
		r.logger.Info(fmt.Sprintf("Start switchover %s with mode: %s and no-wait: %t, current status mode is: %s",
			r.cr.Spec.ElasticsearchDbaasAdapter.Name,
			r.cr.Spec.DisasterRecovery.Mode,
			r.cr.Spec.DisasterRecovery.NoWait,
			r.cr.Status.DisasterRecoveryStatus.Mode))
		if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "active" {
			r.logger.Info(fmt.Sprintf("%s scale-up started", r.cr.Spec.ElasticsearchDbaasAdapter.Name))
			err := r.reconciler.scaleDeploymentWithCheck(r.cr.Spec.ElasticsearchDbaasAdapter.Name, r.cr.Namespace, 1, waitingInterval, scaleTimeout, r.logger)
			if err != nil {
				return err
			}
			r.logger.Info(fmt.Sprintf("%s scale-up completed", r.cr.Spec.ElasticsearchDbaasAdapter.Name))
		} else if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "standby" || strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "disable" {

			r.logger.Info(fmt.Sprintf("%s scale-down started", r.cr.Spec.ElasticsearchDbaasAdapter.Name))
			err := r.reconciler.scaleDeploymentWithCheck(r.cr.Spec.ElasticsearchDbaasAdapter.Name, r.cr.Namespace, 0, waitingInterval, scaleTimeout, r.logger)
			if err != nil {
				return err
			}
			r.logger.Info(fmt.Sprintf("%s scale-down completed", r.cr.Spec.ElasticsearchDbaasAdapter.Name))
		}
		r.logger.Info(fmt.Sprintf("Switchover %s Switchover finished successfully", r.cr.Spec.ElasticsearchDbaasAdapter.Name))
	}
	return nil
}
