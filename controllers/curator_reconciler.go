package controllers

import (
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
	"strings"
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
	if r.cr.Status.DisasterRecoveryStatus.Mode != r.cr.Spec.DisasterRecovery.Mode || r.cr.Status.DisasterRecoveryStatus.Status == "failed" {
		r.logger.Info(fmt.Sprintf("Start switchover %s with mode: %s and no-wait: %t, current status mode is: %s",
			r.cr.Spec.Curator.Name,
			r.cr.Spec.DisasterRecovery.Mode,
			r.cr.Spec.DisasterRecovery.NoWait,
			r.cr.Status.DisasterRecoveryStatus.Mode))
		if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "active" {
			r.logger.Info(fmt.Sprintf("%s scale-up started", r.cr.Spec.Curator.Name))
			err := r.reconciler.scaleDeploymentWithCheck(r.cr.Spec.Curator.Name, r.cr.Namespace, 1, waitingInterval, scaleTimeout, r.logger)
			if err != nil {
				return err
			}
			r.logger.Info(fmt.Sprintf("%s scale-up completed", r.cr.Spec.Curator.Name))
		} else if strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "standby" || strings.ToLower(r.cr.Spec.DisasterRecovery.Mode) == "disable" {

			r.logger.Info(fmt.Sprintf("%s scale-down started", r.cr.Spec.Curator.Name))
			err := r.reconciler.scaleDeploymentWithCheck(r.cr.Spec.Curator.Name, r.cr.Namespace, 0, waitingInterval, scaleTimeout, r.logger)
			if err != nil {
				return err
			}
			r.logger.Info(fmt.Sprintf("%s scale-down completed", r.cr.Spec.Curator.Name))
		}
		r.logger.Info(fmt.Sprintf("Switchover %s Switchover finished successfully", r.cr.Spec.Curator.Name))
	}
	return nil
}
