package controllers

import (
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
	"strings"
)

const (
	dbaasAdapterSecretHashName = "secret.dbaasAdapter"
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
	if r.cr.Spec.DisasterRecovery != nil && r.cr.Status.DisasterRecoveryStatus.Mode != "" {
		r.logger.Info(fmt.Sprintf("Start switchover %s with mode: %s and no-wait: %t, current status mode is: %s",
			r.cr.Spec.DbaasAdapter.Name,
			r.cr.Spec.DisasterRecovery.Mode,
			r.cr.Spec.DisasterRecovery.NoWait,
			r.cr.Status.DisasterRecoveryStatus.Mode))
		if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "active" {
			r.logger.Info(fmt.Sprintf("%s scale-up started", r.cr.Spec.DbaasAdapter.Name))
			err := r.reconciler.scaleDeploymentForDR(r.cr.Spec.DbaasAdapter.Name, r.cr.Namespace, 1, r.cr.Spec.DisasterRecovery.NoWait, r.logger)
			if err != nil {
				return err
			}
			r.logger.Info(fmt.Sprintf("%s scale-up completed", r.cr.Spec.DbaasAdapter.Name))
		} else if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "standby" || strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "disable" {

			r.logger.Info(fmt.Sprintf("%s scale-down started", r.cr.Spec.DbaasAdapter.Name))
			err := r.reconciler.scaleDeploymentForDR(r.cr.Spec.DbaasAdapter.Name, r.cr.Namespace, 0, r.cr.Spec.DisasterRecovery.NoWait, r.logger)
			if err != nil {
				return err
			}
			r.logger.Info(fmt.Sprintf("%s scale-down completed", r.cr.Spec.DbaasAdapter.Name))
		}
		r.logger.Info(fmt.Sprintf("Switchover %s Switchover finished successfully", r.cr.Spec.DbaasAdapter.Name))
	}
	return nil
}
