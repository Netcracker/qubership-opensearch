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
