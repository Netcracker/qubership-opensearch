package controllers

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"github.com/go-logr/logr"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	opensearchConfigHashName = "config.opensearch"
	certificateFilePath      = "/certs/crt.pem"
)

type OpenSearchReconciler struct {
	cr         *opensearchservice.OpenSearchService
	logger     logr.Logger
	reconciler *OpenSearchServiceReconciler
}

func NewOpenSearchReconciler(r *OpenSearchServiceReconciler, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) OpenSearchReconciler {
	return OpenSearchReconciler{
		cr:         cr,
		logger:     logger,
		reconciler: r,
	}
}

func (r OpenSearchReconciler) Reconcile() error {
	opensearchConfigHash, err :=
		r.reconciler.calculateSecretDataHash(r.cr.Spec.OpenSearch.SecurityConfigurationName, opensearchConfigHashName, r.cr, r.logger)
	if err != nil {
		return err
	}

	if r.reconciler.ResourceHashes[opensearchSecretHashName] != "" && r.reconciler.ResourceHashes[opensearchSecretHashName] != opensearchSecretHash ||
		r.reconciler.ResourceHashes[opensearchConfigHashName] != "" && r.reconciler.ResourceHashes[opensearchConfigHashName] != opensearchConfigHash {
		annotations := map[string]string{
			opensearchSecretHashName: opensearchSecretHash,
			opensearchConfigHashName: opensearchConfigHash,
		}

		masterNodeName := r.cr.Name
		if r.cr.Spec.OpenSearch.DedicatedClientPod || r.cr.Spec.OpenSearch.DedicatedDataPod {
			masterNodeName = fmt.Sprintf("%s-master", r.cr.Name)
		}
		if err := r.reconciler.addAnnotationsToStatefulSet(masterNodeName, r.cr.Namespace, annotations, r.logger); err != nil {
			return err
		}

		if r.cr.Spec.OpenSearch.DedicatedClientPod {
			clientNodeName := fmt.Sprintf("%s-client", r.cr.Name)
			if err := r.reconciler.addAnnotationsToDeployment(clientNodeName, r.cr.Namespace, annotations, r.logger); err != nil {
				return err
			}
		}

		if r.cr.Spec.OpenSearch.DedicatedDataPod {
			dataNodeName := fmt.Sprintf("%s-data", r.cr.Name)
			if err := r.reconciler.addAnnotationsToStatefulSet(dataNodeName, r.cr.Namespace, annotations, r.logger); err != nil {
				return err
			}
		}
	}

	r.reconciler.ResourceHashes[opensearchConfigHashName] = opensearchConfigHash
	return nil
}

func (r OpenSearchReconciler) Status() error {
	return nil
}

func (r OpenSearchReconciler) Configure() error {
	if r.cr.Spec.OpenSearch.Snapshots != nil {
		client, err := r.configureClient()
		if err != nil {
			return err
		}
		opensearchCredentials := r.reconciler.parseSecretCredentials(r.cr, r.logger)

		if r.cr.Spec.Curator != nil {
			if err := r.enableCompatibilityMode(client, opensearchCredentials); err != nil {
				return err
			}
		}
		if err := r.createSnapshotsRepository(client, opensearchCredentials, 5); err != nil {
			return err
		}
	}
	return nil
}

// createSnapshotsRepository creates snapshots repository in OpenSearch
func (r *OpenSearchReconciler) createSnapshotsRepository(client http.Client, credentials []string, attemptsNumber int) error {
	r.logger.Info(fmt.Sprintf("Create a snapshot repository with name [%s]", r.cr.Spec.OpenSearch.Snapshots.RepositoryName))
	requestPath := fmt.Sprintf("_snapshot/%s", r.cr.Spec.OpenSearch.Snapshots.RepositoryName)
	requestBody := ""
	if r.cr.Spec.OpenSearch.Snapshots.S3 != nil && r.cr.Spec.OpenSearch.Snapshots.S3.Enabled {
		s3KeyId, s3KeySecret := r.getS3Credentials()
		s3Bucket := r.cr.Spec.OpenSearch.Snapshots.S3.Bucket
		s3Url := r.cr.Spec.OpenSearch.Snapshots.S3.Url
		s3BasePath := r.cr.Spec.OpenSearch.Snapshots.S3.BasePath
		s3Region := r.cr.Spec.OpenSearch.Snapshots.S3.Region
		s3PathStyleAccess := strconv.FormatBool(r.cr.Spec.OpenSearch.Snapshots.S3.PathStyleAccess)
		requestBody = fmt.Sprintf(`{"type": "s3", "settings": {"base_path": "%s", "bucket": "%s", "region": "%s", "endpoint": "%s", "protocol": "http", "access_key": "%s", "secret_key": "%s", "compress": true, "path_style_access": "%s"}}`, s3BasePath, s3Bucket, s3Region, s3Url, s3KeyId, s3KeySecret, s3PathStyleAccess)
	} else {
		requestBody = `{"type": "fs", "settings": {"location": "/usr/share/opensearch/snapshots", "compress": true}}`
	}
	var statusCode int
	var err error
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	for i := 0; i < attemptsNumber; i++ {
		restClient := NewRestClient(url, client, credentials)
		statusCode, _, err = restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(requestBody))
		if err == nil && statusCode == 200 {
			r.logger.Info("Snapshot repository is created")
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("snapshots repository is not created; response status code is %d", statusCode)
}

func (r *OpenSearchReconciler) enableCompatibilityMode(client http.Client, credentials []string) error {
	r.logger.Info("Enable compatibility mode")
	requestPath := "_cluster/settings"
	requestBody := `{"persistent": {"compatibility.override_main_response_version": true}}`
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	restClient := NewRestClient(url, client, credentials)
	statusCode, _, err := restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(requestBody))
	if err == nil && statusCode == 200 {
		r.logger.Info("Compatibility mode is enabled")
		return nil
	}
	return err
}

func (r *OpenSearchReconciler) getS3Credentials() (string, string) {
	secret, err := r.reconciler.findSecret(r.cr.Spec.OpenSearch.Snapshots.S3.SecretName, r.cr.Namespace, r.logger)
	if err != nil {
		r.logger.Info("Can not find s3-credentials secret, use empty user/password")
		return "", ""
	}
	var keyId []byte
	var keySecret []byte
	keyId = secret.Data["s3-key-id"]
	keySecret = secret.Data["s3-key-secret"]
	return string(keyId), string(keySecret)
}

func (r *OpenSearchReconciler) configureClient() (http.Client, error) {
	client := http.Client{}
	if _, err := os.Stat(certificateFilePath); errors.Is(err, os.ErrNotExist) {
		return client, nil
	}
	caCert, err := ioutil.ReadFile(certificateFilePath)
	if err != nil {
		return client, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	return client, nil
}
