package controllers

import (
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
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
	return nil
}
