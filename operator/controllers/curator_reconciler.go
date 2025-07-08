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
	opensearchservice "github.com/Netcracker/opensearch-service/operator/api/v1"
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
