package controllers

import (
	opensearchservice "github.com/Netcracker/qubership-opensearch/operator/api/v1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/util/wait"
	"time"
)

const integrationTestsConditionReason = "OpenSearchIntegrationTestsStatus"

type ReconcileIntegrationTests struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewReconcileIntegrationTests(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService, logger logr.Logger) ReconcileIntegrationTests {
	return ReconcileIntegrationTests{
		reconciler: r,
		cr:         cr,
		logger:     logger,
	}
}

func (r ReconcileIntegrationTests) Reconcile() error {
	return nil
}

func (r ReconcileIntegrationTests) Status() error {
	if !r.cr.Spec.IntegrationTests.WaitForResult {
		return nil
	}

	if err := r.reconciler.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		integrationTestsConditionReason,
		"Start checking OpenSearch Integration Tests")); err != nil {
		return err
	}
	r.logger.Info("Start checking OpenSearch Integration Tests")
	err := wait.PollImmediate(waitingInterval, time.Duration(r.cr.Spec.IntegrationTests.Timeout)*time.Second, func() (done bool, err error) {
		labels := r.getOpenSearchIntegrationTestsLabels()
		return r.reconciler.AreDeploymentsReady(labels, r.cr.Namespace, r.logger), nil
	})
	if err != nil {
		return r.reconciler.updateConditions(NewCondition(statusFalse, typeFailed, integrationTestsConditionReason, "OpenSearch Integration Tests failed. See more details in integration test logs"))
	}
	return r.reconciler.updateConditions(NewCondition(statusTrue, typeReady, integrationTestsConditionReason, "OpenSearch Integration Tests performed successfully"))
}

func (r ReconcileIntegrationTests) getOpenSearchIntegrationTestsLabels() map[string]string {
	return map[string]string{
		"name": r.cr.Spec.IntegrationTests.ServiceName,
	}
}