package controllers

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/pointer"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"
)

const (
	opensearchHttpPort     = 9200
	opensearchHostEnvVar   = "OPENSEARCH_HOST"
	scaleMessageTemplate   = "Timeout occurred during scaling %s"
	httpClientRetryMax     = 3
	scaleTimeout           = 180 * time.Second
	waitingInterval        = 10 * time.Second
	httpClientRetryWaitMax = 10 * time.Second
)

// OpenSearchServiceReconciler reconciles a OpenSearchService object
type OpenSearchServiceReconciler struct {
	client.Client
	Scheme             *runtime.Scheme
	ResourceHashes     map[string]string
	ReplicationWatcher ReplicationWatcher
}

// findSecret returns the secret found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) findSecret(name string, namespace string, logger logr.Logger) (*corev1.Secret, error) {
	logger.Info(fmt.Sprintf("Checking existence of [%s] secret", name))
	foundSecret := &corev1.Secret{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundSecret)
	return foundSecret, err
}

// watchSecret returns the secret found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) watchSecret(secretName string, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) (*corev1.Secret, error) {
	secret, err := r.findSecret(secretName, cr.Namespace, logger)
	if err != nil {
		return nil, err
	} else {
		if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
			return nil, err
		}
		if err := r.updateSecret(secret, logger); err != nil {
			return nil, err
		}
	}
	return secret, nil
}

// updateSecret tries to update specified secret
func (r *OpenSearchServiceReconciler) updateSecret(secret *corev1.Secret, logger logr.Logger) error {
	logger.Info("Updating the found secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
	return r.Client.Update(context.TODO(), secret)
}

// calculateSecretDataHash calculates hash for data of specified secret
func (r *OpenSearchServiceReconciler) calculateSecretDataHash(secretName string, hashName string,
	cr *opensearchservice.OpenSearchService, logger logr.Logger) (string, error) {
	var secret *corev1.Secret
	var err error
	if r.ResourceHashes[hashName] == "" {
		secret, err = r.watchSecret(secretName, cr, logger)
		if err != nil {
			return "", err
		}
	} else {
		secret, err = r.findSecret(secretName, cr.Namespace, logger)
		if err != nil {
			return "", err
		}
	}
	return util.Hash(secret.Data)
}

// parseSecretCredentials gets credentials from OpenSearch secret
func (r *OpenSearchServiceReconciler) parseSecretCredentials(cr *opensearchservice.OpenSearchService, logger logr.Logger) []string {
	secret, err := r.findSecret(fmt.Sprintf("%s-secret", cr.Name), cr.Namespace, logger)
	var credentials []string
	if err == nil {
		user := string(secret.Data["username"])
		password := string(secret.Data["password"])
		if user != "" && password != "" {
			credentials = append(credentials, user, password)
		}
	}
	return credentials
}

// findConfigMap returns the config map found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) findConfigMap(name string, namespace string, logger logr.Logger) (*corev1.ConfigMap, error) {
	logger.Info(fmt.Sprintf("Checking existence of [%s] config map", name))
	foundConfigMap := &corev1.ConfigMap{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundConfigMap)
	return foundConfigMap, err
}

// watchConfigMap returns the config map found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) watchConfigMap(cmName string, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) (*corev1.ConfigMap, error) {
	configMap, err := r.findConfigMap(cmName, cr.Namespace, logger)
	if err != nil {
		return nil, err
	} else {
		if err := controllerutil.SetControllerReference(cr, configMap, r.Scheme); err != nil {
			return nil, err
		}
		if err := r.updateConfigMap(configMap, logger); err != nil {
			return nil, err
		}
	}
	return configMap, nil
}

// updateConfigMap tries to update specified config map
func (r *OpenSearchServiceReconciler) updateConfigMap(configMap *corev1.ConfigMap, logger logr.Logger) error {
	logger.Info("Updating the found config map", "ConfigMap.Namespace", configMap.Namespace, "ConfigMap.Name", configMap.Name)
	return r.Client.Update(context.TODO(), configMap)
}

// calculateConfigDataHash calculates hash for data of specified config map
func (r *OpenSearchServiceReconciler) calculateConfigDataHash(cmName string, hashName string,
	cr *opensearchservice.OpenSearchService, logger logr.Logger) (string, error) {
	var configMap *corev1.ConfigMap
	var err error
	if r.ResourceHashes[hashName] == "" {
		configMap, err = r.watchConfigMap(cmName, cr, logger)
		if err != nil {
			return "", err
		}
	} else {
		configMap, err = r.findConfigMap(cmName, cr.Namespace, logger)
		if err != nil {
			return "", err
		}
	}
	return util.Hash(configMap.Data)
}

// findDeployment returns the deployment found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) findDeployment(name string, namespace string, logger logr.Logger) (*appsv1.Deployment, error) {
	logger.Info(fmt.Sprintf("Checking existence of [%s] deployment", name))
	foundDeployment := &appsv1.Deployment{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundDeployment)
	return foundDeployment, err
}

// updateDeployment tries to update specified deployment
func (r *OpenSearchServiceReconciler) updateDeployment(deployment *appsv1.Deployment, logger logr.Logger) error {
	logger.Info("Updating the deployment",
		"Deployment.Namespace", deployment.Namespace, "Deployment.Name", deployment.Name)
	return r.Client.Update(context.TODO(), deployment)
}

// addAnnotationsToDeployment adds necessary annotations to deployment with specified name and namespace
func (r *OpenSearchServiceReconciler) addAnnotationsToDeployment(name string, namespace string, annotations map[string]string,
	logger logr.Logger) error {
	deployment, err := r.findDeployment(name, namespace, logger)
	if err != nil {
		return err
	}
	if deployment.Spec.Template.Annotations == nil {
		deployment.Spec.Template.Annotations = annotations
	} else {
		for key, value := range annotations {
			deployment.Spec.Template.Annotations[key] = value
		}
	}
	return r.updateDeployment(deployment, logger)
}

// findStatefulSet returns the stateful set found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) findStatefulSet(name string, namespace string, logger logr.Logger) (*appsv1.StatefulSet, error) {
	logger.Info(fmt.Sprintf("Checking existence of [%s] stateful set", name))
	foundStatefulSet := &appsv1.StatefulSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundStatefulSet)
	return foundStatefulSet, err
}

// updateStatefulSet tries to update specified stateful set
func (r *OpenSearchServiceReconciler) updateStatefulSet(statefulSet *appsv1.StatefulSet, logger logr.Logger) error {
	logger.Info("Updating the stateful set",
		"Deployment.Namespace", statefulSet.Namespace, "Deployment.Name", statefulSet.Name)
	return r.Client.Update(context.TODO(), statefulSet)
}

// findService returns the service found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) findService(name string, namespace string, logger logr.Logger) (*corev1.Service, error) {
	logger.Info(fmt.Sprintf("Checking existence of [%s] service", name))
	service := &corev1.Service{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, service)
	return service, err
}

// updateService tries to update specified service
func (r *OpenSearchServiceReconciler) updateService(service *corev1.Service, logger logr.Logger) error {
	logger.Info("Updating the service",
		"Service.Namespace", service.Namespace, "Service.Name", service.Name)
	return r.Client.Update(context.TODO(), service)
}

// addAnnotationsToStatefulSet adds necessary annotations to stateful set with specified name and namespace
func (r *OpenSearchServiceReconciler) addAnnotationsToStatefulSet(name string, namespace string, annotations map[string]string,
	logger logr.Logger) error {
	statefulSet, err := r.findStatefulSet(name, namespace, logger)
	if err != nil {
		return err
	}
	if statefulSet.Spec.Template.Annotations == nil {
		statefulSet.Spec.Template.Annotations = annotations
	} else {
		for key, value := range annotations {
			statefulSet.Spec.Template.Annotations[key] = value
		}
	}
	return r.updateStatefulSet(statefulSet, logger)
}

func (r *OpenSearchServiceReconciler) scaleDeployment(name string, namespace string, replicas int32, logger logr.Logger) error {
	deployment, err := r.findDeployment(name, namespace, logger)
	if err == nil {
		deployment.Spec.Replicas = pointer.Int32Ptr(replicas)
		err := r.Client.Update(context.TODO(), deployment)
		return err
	}
	return err
}

func (r *OpenSearchServiceReconciler) scaleDeploymentWithCheck(name string, namespace string, replicas int32, interval, timeout time.Duration, logger logr.Logger) error {
	err := r.scaleDeployment(name, namespace, replicas, logger)
	if err != nil {
		logger.Error(err, "Deployment update failed")
		return err
	}
	logger.Info(fmt.Sprintf("deployment %s scaled", name))
	err = wait.PollImmediate(interval, timeout, func() (done bool, err error) {
		return r.isDeploymentReady(name, namespace, logger), nil
	})
	if err != nil {
		direction := "up"
		if replicas == 0 {
			direction = "down"
		}
		logger.Error(err, fmt.Sprintf(scaleMessageTemplate, direction))
		return err
	}
	return nil
}

func (r *OpenSearchServiceReconciler) scaleDeploymentForNoWait(name string, namespace string, replicas int32, noWait bool, logger logr.Logger) error {
	if noWait {
		return r.scaleDeployment(name, namespace, replicas, logger)
	} else {
		return r.scaleDeploymentWithCheck(name, namespace, replicas, waitingInterval, scaleTimeout, logger)
	}
}

func (r *OpenSearchServiceReconciler) scaleDeploymentForDR(name string, cr *opensearchservice.OpenSearchService, logger logr.Logger) error {
	if cr.Spec.DisasterRecovery != nil && cr.Status.DisasterRecoveryStatus.Mode != "" {
		logger.Info(fmt.Sprintf("Start switchover %s with mode: %s and no-wait: %t, current status mode is: %s",
			name,
			cr.Spec.DisasterRecovery.Mode,
			cr.Spec.DisasterRecovery.NoWait,
			cr.Status.DisasterRecoveryStatus.Mode))
		if strings.ToLower(cr.Spec.DisasterRecovery.Mode) == "active" {
			logger.Info(fmt.Sprintf("%s scale-up started", name))
			err := r.scaleDeploymentForNoWait(name, cr.Namespace, 1, cr.Spec.DisasterRecovery.NoWait, logger)
			if err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("%s scale-up completed", name))
		} else if strings.ToLower(cr.Spec.DisasterRecovery.Mode) == "standby" || strings.ToLower(cr.Spec.DisasterRecovery.Mode) == "disable" {

			logger.Info(fmt.Sprintf("%s scale-down started", name))
			err := r.scaleDeploymentForNoWait(name, cr.Namespace, 0, cr.Spec.DisasterRecovery.NoWait, logger)
			if err != nil {
				return err
			}
			logger.Info(fmt.Sprintf("%s scale-down completed", name))
		}
		logger.Info(fmt.Sprintf("Switchover %s finished successfully", name))
	}
	return nil
}

func (r *OpenSearchServiceReconciler) isDeploymentReady(deploymentName string, namespace string, logger logr.Logger) bool {
	deployment, err := r.findDeployment(deploymentName, namespace, logger)
	if err != nil {
		logger.Error(err, "Cannot check deployment status")
		return false
	}
	availableReplicas := util.Min(deployment.Status.ReadyReplicas, deployment.Status.UpdatedReplicas)
	return *deployment.Spec.Replicas == availableReplicas
}

// disableClientService disables OpenSearch client service
func (r *OpenSearchServiceReconciler) disableClientService(name string, namespace string, logger logr.Logger) error {
	service, err := r.findService(name, namespace, logger)
	if err != nil {
		return err
	}
	service.Spec.Selector["none"] = "true"
	return r.updateService(service, logger)
}

// enableClientService enables OpenSearch client service
func (r *OpenSearchServiceReconciler) enableClientService(name string, namespace string, logger logr.Logger) error {
	service, err := r.findService(name, namespace, logger)
	if err != nil {
		return err
	}
	delete(service.Spec.Selector, "none")
	return r.updateService(service, logger)
}

func (r *OpenSearchServiceReconciler) createUrl(host string, port int) string {
	// if OpenSearch host specified, you can connect to operator remotely
	osHost := os.Getenv(opensearchHostEnvVar)
	if osHost != "" {
		return osHost
	}
	protocol := "https"
	if _, err := os.Stat(certificateFilePath); errors.Is(err, os.ErrNotExist) {
		protocol = "http"
	}
	return fmt.Sprintf("%s://%s-internal:%d", protocol, host, port)
}

func (r *OpenSearchServiceReconciler) createHttpClient() http.Client {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = httpClientRetryMax
	retryClient.RetryWaitMax = httpClientRetryWaitMax
	return *retryClient.StandardClient()
}

func (r *OpenSearchServiceReconciler) configureClient() (http.Client, error) {
	httpClient := r.createHttpClient()
	if _, err := os.Stat(certificateFilePath); errors.Is(err, os.ErrNotExist) {
		return httpClient, nil
	}
	caCert, err := ioutil.ReadFile(certificateFilePath)
	if err != nil {
		return httpClient, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	httpClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	return httpClient, nil
}
