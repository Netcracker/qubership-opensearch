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
	opensearchservice "github.com/Netcracker/opensearch-service/api/v1"
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
