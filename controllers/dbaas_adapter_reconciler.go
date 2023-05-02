package controllers

import (
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
)

const (
	dbaasAdapterSecretHashName = "secret.dbaasAdapter"
	dbaasCertificateFilePath   = "/certs/dbaas-adapter/crt.pem"
)

type DbaasAdapterReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewDbaasAdapterReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) DbaasAdapterReconciler {
	return DbaasAdapterReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r DbaasAdapterReconciler) Reconcile() error {
	dbaasAdapterSecretHash, err :=
		r.reconciler.calculateSecretDataHash(r.cr.Spec.DbaasAdapter.SecretName, dbaasAdapterSecretHashName, r.cr, r.logger)
	if err != nil {
		return err
	}
	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[dbaasAdapterSecretHashName] != "" && r.reconciler.ResourceHashes[dbaasAdapterSecretHashName] != dbaasAdapterSecretHash {
		annotations := map[string]string{
			opensearchSecretHashName:   opensearchSecretHash,
			dbaasAdapterSecretHashName: dbaasAdapterSecretHash,
		}

		if err := r.reconciler.addAnnotationsToDeployment(r.cr.Spec.DbaasAdapter.Name, r.cr.Namespace, annotations, r.logger); err != nil {
			return err
		}
	}

	r.reconciler.ResourceHashes[dbaasAdapterSecretHashName] = dbaasAdapterSecretHash
	return nil
}

func (r DbaasAdapterReconciler) Status() error {
	return nil
}

func (r DbaasAdapterReconciler) Configure() error {
	return r.reconciler.scaleDeploymentForDR(r.cr.Spec.DbaasAdapter.Name, r.cr, r.logger)
}
