package controllers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/util/wait"
	"net/http"
	"strconv"
	"strings"
	"time"

	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
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
	return nil
}

func (r OpenSearchReconciler) Status() error {
	return nil
}

func (r OpenSearchReconciler) Configure() error {
	restClient, err := r.processSecurity()
	if err != nil {
		return err
	}

	if r.cr.Spec.OpenSearch.Snapshots != nil {
		if err = r.createSnapshotsRepository(restClient, 5); err != nil {
			return err
		}
	}

	if r.cr.Spec.OpenSearch.CompatibilityModeEnabled {
		if err = r.enableCompatibilityMode(restClient); err != nil {
			return err
		}
	}
	return nil
}

func (r OpenSearchReconciler) processSecurity() (*util.RestClient, error) {
	restClient, err := r.updateCredentials()
	if err != nil {
		if strings.Contains(err.Error(), "is read-only") {
			clusterManagerPod, requestErr := r.getClusterManagerNode(restClient)
			if requestErr != nil {
				return restClient, requestErr
			}
			commandErr := r.reconciler.runCommandInPod(clusterManagerPod, "opensearch", r.cr.Namespace,
				[]string{"/bin/sh", "-c", "/usr/share/opensearch/bin/reconfiguration.sh"})
			if commandErr != nil {
				return restClient, commandErr
			}
		}
		return restClient, err
	}

	opensearchConfigHash, err :=
		r.reconciler.calculateSecretDataHash(r.cr.Spec.OpenSearch.SecurityConfigurationName, opensearchConfigHashName, r.cr, r.logger)
	if err != nil {
		return restClient, err
	}
	if r.reconciler.ResourceHashes[opensearchConfigHashName] != "" && r.reconciler.ResourceHashes[opensearchConfigHashName] != opensearchConfigHash {
		err := r.updateSecurityConfiguration(restClient)
		if err != nil {
			return restClient, err
		}
	}
	r.reconciler.ResourceHashes[opensearchConfigHashName] = opensearchConfigHash
	return restClient, nil
}

func (r OpenSearchReconciler) updateCredentials() (*util.RestClient, error) {
	url := r.reconciler.createUrl(r.cr.Name, opensearchHttpPort)
	client, err := r.reconciler.configureClient()
	if err != nil {
		return nil, err
	}
	oldCredentials := r.reconciler.parseSecretCredentials(fmt.Sprintf(oldSecretPattern, r.cr.Name), r.cr.Namespace, r.logger)
	newCredentials := r.reconciler.parseSecretCredentials(fmt.Sprintf(secretPattern, r.cr.Name), r.cr.Namespace, r.logger)
	restClient := util.NewRestClient(url, client, oldCredentials)

	if newCredentials.Username != oldCredentials.Username ||
		newCredentials.Password != oldCredentials.Password {
		if newCredentials.Username != oldCredentials.Username {
			if err = r.createNewUser(newCredentials.Username, newCredentials.Password, restClient); err != nil {
				return restClient, err
			}
			if err = r.removeUser(oldCredentials.Username, restClient); err != nil {
				return restClient, err
			}
		} else {
			if err = r.changeUserPassword(newCredentials.Username, newCredentials.Password, restClient); err != nil {
				return restClient, err
			}
		}
		err = wait.PollImmediate(waitingInterval, updateTimeout, func() (bool, error) {
			err = r.reconciler.updateSecretWithCredentials(fmt.Sprintf(oldSecretPattern, r.cr.Name), r.cr.Namespace, newCredentials, r.logger)
			if err != nil {
				r.logger.Error(err, "Unable to update secret with credentials")
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			return restClient, err
		}
		restClient = util.NewRestClient(url, client, newCredentials)
	}
	return restClient, nil
}

func (r OpenSearchReconciler) getClusterManagerNode(restClient *util.RestClient) (string, error) {
	requestPath := "_cat/cluster_manager?h=node"
	statusCode, responseBody, err := restClient.SendBasicRequest(http.MethodGet, requestPath, nil, false)
	if err == nil {
		if statusCode == http.StatusOK {
			return strings.TrimSpace(string(responseBody)), nil
		}
		return "", fmt.Errorf("unable to receive cluster_manager node: [%d] %s", statusCode, responseBody)
	}
	return "", err
}

func (r OpenSearchReconciler) createNewUser(username string, password string, restClient *util.RestClient) error {
	if username == "" || password == "" {
		r.logger.Error(nil, "Unable to create user with empty name or password")
		return nil
	}
	requestPath := fmt.Sprintf("_plugins/_security/api/internalusers/%s", username)
	body := fmt.Sprintf(`{"password": "%s", "description": "Admin user", "backend_roles": ["admin"], 
"opendistro_security_roles": ["all_access", "manage_snapshots"]}`, password)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK || statusCode == http.StatusCreated {
			r.logger.Info("The user is successfully created")
			return nil
		}
		return fmt.Errorf("user creation went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

func (r OpenSearchReconciler) changeUserPassword(username string, password string, restClient *util.RestClient) error {
	if username == "" || password == "" {
		r.logger.Error(nil, "Unable to update user with empty name or password")
		return nil
	}
	requestPath := fmt.Sprintf("_plugins/_security/api/internalusers/%s", username)
	body := fmt.Sprintf(`[{"op": "add", "path": "/password", "value": "%s"}]`, password)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPatch, requestPath, strings.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK {
			r.logger.Info("The password for user is successfully updated")
			return nil
		}
		return fmt.Errorf("user update went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

func (r OpenSearchReconciler) removeUser(username string, restClient *util.RestClient) error {
	if username == "" {
		return nil
	}
	requestPath := fmt.Sprintf("_plugins/_security/api/internalusers/%s", username)
	statusCode, responseBody, err := restClient.SendRequest(http.MethodDelete, requestPath, nil)
	if err == nil {
		if statusCode == http.StatusOK || statusCode == http.StatusNotFound {
			r.logger.Info("The user is successfully deleted")
			return nil
		}
		return fmt.Errorf("user removal went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

// updateSecurityConfiguration updates security configuration in OpenSearch
func (r OpenSearchReconciler) updateSecurityConfiguration(restClient *util.RestClient) error {
	secret, err := r.reconciler.findSecret(r.cr.Spec.OpenSearch.SecurityConfigurationName, r.cr.Namespace, r.logger)
	if err != nil {
		return err
	}
	securityConfiguration := secret.Data["config.yml"]
	if securityConfiguration == nil {
		r.logger.Info("Security configuration is empty, so there is nothing to update")
		return nil
	}
	var configuration map[string]interface{}
	if err = yaml.Unmarshal(securityConfiguration, &configuration); err != nil {
		return err
	}
	if configuration["config"] == nil {
		r.logger.Info("Security configuration is empty, so there is nothing to update")
		return nil
	}
	return r.updateSecurityConfig(configuration["config"], restClient)
}

func (r OpenSearchReconciler) updateSecurityConfig(configuration interface{}, restClient *util.RestClient) error {
	body, err := json.Marshal(configuration)
	if err != nil {
		return err
	}
	requestPath := "_plugins/_security/api/securityconfig/config"
	statusCode, responseBody, err := restClient.SendRequest(http.MethodPut, requestPath, bytes.NewReader(body))
	if err == nil {
		if statusCode == http.StatusOK {
			r.logger.Info("Security configuration is successfully updated")
			return nil
		}
		return fmt.Errorf("security configuration update went wrong: [%d] %s", statusCode, responseBody)
	}
	return err
}

// createSnapshotsRepository creates snapshots repository in OpenSearch
func (r OpenSearchReconciler) createSnapshotsRepository(restClient *util.RestClient, attemptsNumber int) error {
	r.logger.Info(fmt.Sprintf("Create a snapshot repository with name [%s]", r.cr.Spec.OpenSearch.Snapshots.RepositoryName))
	requestPath := fmt.Sprintf("_snapshot/%s", r.cr.Spec.OpenSearch.Snapshots.RepositoryName)
	requestBody := r.getSnapshotsRepositoryBody()
	var statusCode int
	var err error
	for i := 0; i < attemptsNumber; i++ {
		statusCode, _, err = restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(requestBody))
		if err == nil && statusCode == http.StatusOK {
			r.logger.Info("Snapshot repository is created")
			return nil
		}
		time.Sleep(5 * time.Second)
	}
	return fmt.Errorf("snapshots repository is not created; response status code is %d", statusCode)
}

func (r OpenSearchReconciler) enableCompatibilityMode(restClient *util.RestClient) error {
	r.logger.Info("Enable compatibility mode")
	requestPath := "_cluster/settings"
	requestBody := `{"persistent": {"compatibility.override_main_response_version": true}}`
	statusCode, _, err := restClient.SendRequest(http.MethodPut, requestPath, strings.NewReader(requestBody))
	if err == nil && statusCode == http.StatusOK {
		r.logger.Info("Compatibility mode is enabled")
		return nil
	}
	return err
}

func (r OpenSearchReconciler) getS3Credentials() (string, string) {
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

func (r OpenSearchReconciler) getSnapshotsRepositoryBody() string {
	if r.cr.Spec.OpenSearch.Snapshots.S3 != nil {
		if r.cr.Spec.OpenSearch.Snapshots.S3.GcsEnabled {
			s3Bucket := r.cr.Spec.OpenSearch.Snapshots.S3.Bucket
			return fmt.Sprintf(`{"type": "gcs", "settings": {"bucket": "%s", "client": "default"}}`, s3Bucket)
		}
		if r.cr.Spec.OpenSearch.Snapshots.S3.Enabled {
			s3KeyId, s3KeySecret := r.getS3Credentials()
			s3Bucket := r.cr.Spec.OpenSearch.Snapshots.S3.Bucket
			s3Url := r.cr.Spec.OpenSearch.Snapshots.S3.Url
			s3BasePath := r.cr.Spec.OpenSearch.Snapshots.S3.BasePath
			s3Region := r.cr.Spec.OpenSearch.Snapshots.S3.Region
			s3PathStyleAccess := strconv.FormatBool(r.cr.Spec.OpenSearch.Snapshots.S3.PathStyleAccess)
			return fmt.Sprintf(`{"type": "s3", "settings": {"base_path": "%s", "bucket": "%s", "region": "%s", "endpoint": "%s", "protocol": "http", "access_key": "%s", "secret_key": "%s", "compress": true, "path_style_access": "%s"}}`, s3BasePath, s3Bucket, s3Region, s3Url, s3KeyId, s3KeySecret, s3PathStyleAccess)
		}
	}
	return `{"type": "fs", "settings": {"location": "/usr/share/opensearch/snapshots", "compress": true}}`
}
