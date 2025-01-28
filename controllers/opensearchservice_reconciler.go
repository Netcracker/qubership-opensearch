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
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	opensearchservice "github.com/Netcracker/opensearch-service/api/v1"
	"github.com/Netcracker/opensearch-service/util"
	"github.com/go-logr/logr"
	"github.com/hashicorp/go-retryablehttp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/utils/pointer"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	kubeconfig "sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strings"
	"time"
)

const (
	opensearchHttpPort     = 9200
	opensearchHostEnvVar   = "OPENSEARCH_HOST"
	scaleMessageTemplate   = "Timeout occurred during scaling %s"
	httpClientRetryMax     = 2
	scaleTimeout           = 180 * time.Second
	updateTimeout          = 60 * time.Second
	waitingInterval        = 10 * time.Second
	httpClientRetryWaitMax = 5 * time.Second
	httpClientTimeout      = 60 * time.Second
	secretPattern          = "%s-secret"
	oldSecretPattern       = "%s-secret-old"
)

// OpenSearchServiceReconciler reconciles a OpenSearchService object
type OpenSearchServiceReconciler struct {
	client.Client
	Scheme                *runtime.Scheme
	ResourceHashes        map[string]string
	ReplicationWatcher    ReplicationWatcher
	SlowLogIndicesWatcher SlowLogIndicesWatcher
	StatusUpdater         util.StatusUpdater
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
		// Check if there's an existing owner reference
		if existing := metav1.GetControllerOf(secret); existing != nil && !referSameObject(existing.Name, existing.APIVersion, existing.Kind, cr.Name, cr.APIVersion, cr.Kind) {
			secret.OwnerReferences = nil
		}
		if err := controllerutil.SetControllerReference(cr, secret, r.Scheme); err != nil {
			return nil, err
		}
		if err := r.updateSecret(secret, logger); err != nil {
			return nil, err
		}
	}
	return secret, nil
}

func referSameObject(aName string, aGroup string, aKind string, bName string, bGroup string, bKind string) bool {
	aGV, err := schema.ParseGroupVersion(aGroup)
	if err != nil {
		return false
	}

	bGV, err := schema.ParseGroupVersion(bGroup)
	if err != nil {
		return false
	}

	return aGV.Group == bGV.Group && aKind == bKind && aName == bName
}

// updateSecret tries to update specified secret
func (r *OpenSearchServiceReconciler) updateSecret(secret *corev1.Secret, logger logr.Logger) error {
	logger.Info("Updating the found secret", "Secret.Namespace", secret.Namespace, "Secret.Name", secret.Name)
	return r.Client.Update(context.TODO(), secret)
}

// updateSecretWithCredentials tries to update specified secret with specified credentials
func (r *OpenSearchServiceReconciler) updateSecretWithCredentials(name string, namespace string,
	credentials util.Credentials, logger logr.Logger) error {
	secret, err := r.findSecret(name, namespace, logger)
	if err != nil {
		return err
	}
	secret.StringData = map[string]string{
		"username": credentials.Username,
		"password": credentials.Password,
	}
	return r.updateSecret(secret, logger)
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

// parseOpenSearchCredentials gets credentials from OpenSearch secret
func (r *OpenSearchServiceReconciler) parseOpenSearchCredentials(cr *opensearchservice.OpenSearchService, logger logr.Logger) util.Credentials {
	if cr.Spec.ExternalOpenSearch != nil {
		return r.parseSecretCredentials(fmt.Sprintf(secretPattern, cr.Name), cr.Namespace, logger)
	}
	return r.parseSecretCredentials(fmt.Sprintf(oldSecretPattern, cr.Name), cr.Namespace, logger)
}

// parseSecretCredentials gets credentials from specified secret
func (r *OpenSearchServiceReconciler) parseSecretCredentials(name string, namespace string, logger logr.Logger) util.Credentials {
	return r.parseSecretCredentialsByKeys(name, namespace, "username", "password", logger)
}

func (r *OpenSearchServiceReconciler) parseSecretCredentialsByKeys(name string, namespace string, usernameKey string,
	passwordKey string, logger logr.Logger) util.Credentials {
	var credentials util.Credentials
	secret, err := r.findSecret(name, namespace, logger)
	if err == nil {
		username := string(secret.Data[usernameKey])
		password := string(secret.Data[passwordKey])
		if username != "" && password != "" {
			credentials = util.NewCredentials(username, password)
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
		if existing := metav1.GetControllerOf(configMap); existing != nil && !referSameObject(existing.Name, existing.APIVersion, existing.Kind, cr.Name, cr.APIVersion, cr.Kind) {
			configMap.OwnerReferences = nil
		}
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

// runCommandInPod runs specified command in specified pod
func (r *OpenSearchServiceReconciler) runCommandInPod(podName string, container string, namespace string, command []string) error {
	config := kubeconfig.GetConfigOrDie()
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	request := kubeClient.CoreV1().RESTClient().Post().
		Namespace(namespace).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Stdout:    true,
			Stderr:    true,
			Container: container,
			Command:   command,
		}, scheme.ParameterCodec)
	executor, err := remotecommand.NewSPDYExecutor(config, "POST", request.URL())
	if err != nil {
		return fmt.Errorf("unable execute command via SPDY: %+v", err)
	}
	var execOut bytes.Buffer
	var execErr bytes.Buffer
	err = executor.Stream(remotecommand.StreamOptions{
		Stdout: &execOut,
		Stderr: &execErr,
		Tty:    false,
	})
	if err != nil {
		if strings.Contains(err.Error(), "command terminated with exit code 127") {
			return fmt.Errorf("the command %v is not found in '%s' pod", command, podName)
		}
		return fmt.Errorf("there is a problem during command execution: %+v", err)
	}
	if execErr.Len() > 0 {
		return fmt.Errorf("there is a problem during command execution: %s", execErr.String())
	}
	log.Info(fmt.Sprintf("Executed command output: %s", execOut.String()))
	return nil
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

func (r *OpenSearchServiceReconciler) scaleDeployment(name string, namespace string, replicas int32, logger logr.Logger) error {
	deployment, err := r.findDeployment(name, namespace, logger)
	if err == nil {
		deployment.Spec.Replicas = pointer.Int32Ptr(replicas)
		return r.Client.Update(context.TODO(), deployment)
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
	err = wait.Poll(interval, timeout, func() (done bool, err error) {
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
	retryClient.HTTPClient = &http.Client{Timeout: httpClientTimeout}
	retryClient.RetryMax = httpClientRetryMax
	retryClient.RetryWaitMax = httpClientRetryWaitMax
	return *retryClient.StandardClient()
}

func (r *OpenSearchServiceReconciler) configureClient() (http.Client, error) {
	return r.configureClientWithCertificate(certificateFilePath)
}

// configureClientWithCertificate configures client with certificates from specified file
func (r *OpenSearchServiceReconciler) configureClientWithCertificate(certificatePath string) (http.Client, error) {
	httpClient := r.createHttpClient()
	if _, err := os.Stat(certificatePath); errors.Is(err, os.ErrNotExist) {
		return httpClient, nil
	}
	caCert, err := os.ReadFile(certificatePath)
	if err != nil {
		log.Error(err, fmt.Sprintf("Unable to read certificates from %s file", certificatePath))
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

// findStatefulSet returns the stateful set found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) findStatefulSet(name string, namespace string, logger logr.Logger) (*appsv1.StatefulSet, error) {
	logger.Info(fmt.Sprintf("Checking existence of [%s] StatefulSet", name))
	foundStatefulSet := &appsv1.StatefulSet{}
	err := r.Client.Get(context.TODO(), types.NamespacedName{Name: name, Namespace: namespace}, foundStatefulSet)
	return foundStatefulSet, err
}

func (r *OpenSearchServiceReconciler) findPod(name string, namespace string, logger logr.Logger) (*corev1.Pod, error) {
	pod := &corev1.Pod{}
	podInfo := types.NamespacedName{Name: name, Namespace: namespace}
	err := r.Client.Get(context.TODO(), podInfo, pod)
	return pod, err
}

// updateStatefulSet tries to update specified stateful set
func (r *OpenSearchServiceReconciler) updateStatefulSet(statefulSet *appsv1.StatefulSet, logger logr.Logger) error {
	logger.Info("Updating the found StatefulSet", "StatefulSet.Namespace", statefulSet.Namespace, "StatefulSet.Name", statefulSet.Name)
	return r.Client.Update(context.TODO(), statefulSet)
}

// watchStatefulSet returns the stateful set found by name and namespace and error if it occurred
func (r *OpenSearchServiceReconciler) watchStatefulSet(setName string, cr *opensearchservice.OpenSearchService,
	logger logr.Logger) (*appsv1.StatefulSet, error) {
	logger.Info(fmt.Sprintf("Try to start watching %s StatefulSet ...", setName))

	statefulSet, err := r.findStatefulSet(setName, cr.Namespace, logger)
	if err != nil {
		logger.Error(err, fmt.Sprintf("Error while searching %s StatefulSet", setName))
		return nil, err
	}
	if existing := metav1.GetControllerOf(statefulSet); existing != nil && !referSameObject(existing.Name, existing.APIVersion, existing.Kind, cr.Name, cr.APIVersion, cr.Kind) {
		statefulSet.OwnerReferences = nil
	}
	if err := controllerutil.SetControllerReference(cr, statefulSet, r.Scheme); err != nil {
		logger.Error(err, fmt.Sprintf("Error while set controller owner reference to  %s StatefulSet", setName))
		return nil, err
	}

	if err := r.updateStatefulSet(statefulSet, logger); err != nil {
		logger.Error(err, fmt.Sprintf("Error while updating %s StatefulSet", setName))
		return nil, err
	}

	return statefulSet, nil
}

func (r *OpenSearchServiceReconciler) deletePodByName(podName string, namespace string, logger logr.Logger) error {
	logger.Info(fmt.Sprintf("Deleting pod %s in %s namespace", podName, namespace))

	pod, err := r.findPod(podName, namespace, logger)
	if err != nil {
		return err
	}

	if err := r.Delete(context.TODO(), pod); err != nil {
		logger.Error(err, fmt.Sprintf("Pod %s was not deleted", podName))
		return err
	}

	return nil
}
