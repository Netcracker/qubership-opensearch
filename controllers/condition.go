package controllers

import (
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	statusFalse    = "False"
	statusTrue     = "True"
	typeInProgress = "In progress"
	typeFailed     = "Failed"
	typeSuccessful = "Successful"
)

func NewCondition(conditionStatus string, conditionType string, conditionReason string, conditionMessage string) opensearchservice.StatusCondition {
	return opensearchservice.StatusCondition{
		Type:    conditionType,
		Status:  conditionStatus,
		Reason:  conditionReason,
		Message: conditionMessage,
	}
}

func (r *OpenSearchServiceReconciler) updateConditions(condition opensearchservice.StatusCondition) error {
	return r.StatusUpdater.UpdateStatusWithRetry(func(instance *opensearchservice.OpenSearchService) {
		currentConditions := instance.Status.Conditions
		condition.LastTransitionTime = metav1.Now().String()
		currentConditions = addCondition(currentConditions, condition)
		instance.Status.Conditions = currentConditions
	})
}

func addCondition(currentConditions []opensearchservice.StatusCondition, condition opensearchservice.StatusCondition) []opensearchservice.StatusCondition {
	for i, currentCondition := range currentConditions {
		if currentCondition.Reason == condition.Reason {
			if currentCondition.Type != condition.Type ||
				currentCondition.Status != condition.Status ||
				currentCondition.Message != condition.Message {
				currentConditions[i] = condition
			}
			return currentConditions
		}
	}
	return append(currentConditions, condition)
}
