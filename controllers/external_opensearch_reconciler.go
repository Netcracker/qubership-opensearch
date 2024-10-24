package controllers

import (
	"encoding/json"
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	"net/http"
	"strings"
)

const (
	externalSpecHashName = "spec.externalOpenSearch"
)

type ExternalOpenSearchReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewExternalOpenSearchReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) ExternalOpenSearchReconciler {
	return ExternalOpenSearchReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r ExternalOpenSearchReconciler) Reconcile() error {
	externalOpenSearchSpecHash, err := util.Hash(r.cr.Spec.ExternalOpenSearch)
	if err != nil {
		return err
	}
	if r.reconciler.ResourceHashes[externalSpecHashName] != externalOpenSearchSpecHash {
		if err := r.performExternalOpenSearchConfiguration(); err != nil {
			return err
		}
	}
	r.reconciler.ResourceHashes[externalSpecHashName] = externalOpenSearchSpecHash
	return nil
}

func (r ExternalOpenSearchReconciler) Status() error {
	return nil
}

func (r ExternalOpenSearchReconciler) Configure() error {
	return nil
}

func (r ExternalOpenSearchReconciler) performExternalOpenSearchConfiguration() error {
	client, _ := r.reconciler.configureClient()
	credentials := r.reconciler.parseSecretCredentials(fmt.Sprintf(secretPattern, r.cr.Name), r.cr.Namespace, r.logger)
	restClient := util.NewRestClient(r.cr.Spec.ExternalOpenSearch.Url, client, credentials)
	jsonConfig, err := json.Marshal(r.cr.Spec.ExternalOpenSearch.Config)
	if err != nil {
		return err
	}
	body := fmt.Sprintf(`{"persistent":%s}`, jsonConfig)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPut, "_cluster/settings", strings.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK {
			r.logger.Info(fmt.Sprintf("The settings [%s] is successfully applied", jsonConfig))
			return nil
		}
		return fmt.Errorf("setting applying went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}
