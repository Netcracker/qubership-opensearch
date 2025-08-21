// Copyright 2024-2025 NetCracker Technology Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controllers

import (
	opensearchservice "github.com/Netcracker/qubership-opensearch/operator/api/v1"
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
