package util

import (
	"context"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type StatusUpdater struct {
	client    client.Client
	name      string
	namespace string
}

func NewStatusUpdater(client client.Client, cr *opensearchservice.OpenSearchService) StatusUpdater {
	return StatusUpdater{
		client:    client,
		name:      cr.Name,
		namespace: cr.Namespace,
	}
}

func (su StatusUpdater) UpdateStatusWithRetry(statusUpdateFunc func(*opensearchservice.OpenSearchService)) error {
	return retry.RetryOnConflict(retry.DefaultRetry, func() error {
		instance := &opensearchservice.OpenSearchService{}
		if err := su.client.Get(context.TODO(),
			types.NamespacedName{Name: su.name, Namespace: su.namespace}, instance); err != nil {
			return err
		}
		statusUpdateFunc(instance)
		return su.client.Status().Update(context.TODO(), instance)
	})
}
