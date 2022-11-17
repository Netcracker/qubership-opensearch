package controllers

import (
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
)

const (
	curatorSecretHashName = "secret.curator"
)

type CuratorReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewCuratorReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) CuratorReconciler {
	return CuratorReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r CuratorReconciler) Reconcile() error {
	curatorSecretHash, err :=
		r.reconciler.calculateSecretDataHash(r.cr.Spec.Curator.SecretName, curatorSecretHashName, r.cr, r.logger)
	if err != nil {
		return err
	}

	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[curatorSecretHashName] != "" && r.reconciler.ResourceHashes[curatorSecretHashName] != curatorSecretHash {
		annotations := map[string]string{
			opensearchSecretHashName: opensearchSecretHash,
			curatorSecretHashName:    curatorSecretHash,
		}

		if err := r.reconciler.addAnnotationsToDeployment(r.cr.Spec.Curator.Name, r.cr.Namespace, annotations, r.logger); err != nil {
			return err
		}
	}

	r.reconciler.ResourceHashes[curatorSecretHashName] = curatorSecretHash
	return nil
}

func (r CuratorReconciler) Status() error {
	return nil
}

func (r CuratorReconciler) Configure() error {
	return r.reconciler.scaleDeploymentForDR(r.cr.Spec.Curator.Name, r.cr, r.logger)
}
