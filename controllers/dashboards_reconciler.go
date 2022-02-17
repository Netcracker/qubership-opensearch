package controllers

import (
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
)

const (
	dashboardsSecretHashName = "secret.dashboards"
)

type DashboardsReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewDashboardsReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) DashboardsReconciler {
	return DashboardsReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r DashboardsReconciler) Reconcile() error {
	var dashboardsSecretHash string
	if r.cr.Spec.Dashboards.SecretName != "" {
		var err error
		dashboardsSecretHash, err =
			r.reconciler.calculateSecretDataHash(r.cr.Spec.Dashboards.SecretName, dashboardsSecretHashName, r.cr, r.logger)
		if err != nil {
			return err
		}
	}

	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[dashboardsSecretHashName] != "" && r.reconciler.ResourceHashes[dashboardsSecretHashName] != dashboardsSecretHash {
		annotations := map[string]string{
			opensearchSecretHashName: opensearchSecretHash,
			dashboardsSecretHashName: dashboardsSecretHash,
		}

		if err := r.reconciler.addAnnotationsToDeployment(r.cr.Spec.Dashboards.Name, r.cr.Namespace, annotations, r.logger); err != nil {
			return err
		}
	}

	r.reconciler.ResourceHashes[dashboardsSecretHashName] = dashboardsSecretHash
	return nil
}

func (r DashboardsReconciler) Status() error {
	return nil
}

func (r DashboardsReconciler) Configure() error {
	return nil
}
