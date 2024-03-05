package controllers

import (
	"context"
	"fmt"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/util/workqueue"
	"net/http"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"

	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
)

const (
	defaultReadinessTimeout          = 800 * time.Second
	maxRateLimiterDelay              = 60
	minRateLimiterDelay              = 5
	opensearchSecretHashName         = "secret.opensearch"
	opensearchServiceConditionReason = "ReconcileCycleStatus"
)

var opensearchSecretHash = ""

var log = logf.Log.WithName("controller_opensearchservice")

type ReconcileService interface {
	Reconcile() error
	Status() error
	Configure() error
}

type NotReadyError struct {
	StatusCode int
	Err        error
}

func (nre NotReadyError) Error() string {
	message := "OpenSearch is not ready yet!"
	if nre.StatusCode > 0 {
		message = fmt.Sprintf("%s Status code - [%d].", message, nre.StatusCode)
	}
	if nre.Err != nil {
		message = fmt.Sprintf("%s Error - [%s].", message, nre.Err.Error())
	}
	return message
}

//+kubebuilder:rbac:groups=netcracker.com,resources=opensearchservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=netcracker.com,resources=opensearchservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=netcracker.com,resources=opensearchservices/finalizers,verbs=update

func (r *OpenSearchServiceReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling OpenSearch service")

	//TODO: implement channel communication between DR server goroutine and current goroutine instead of sleeping
	time.Sleep(time.Second * 5)

	// Fetch the OpenSearchService instance
	instance := &opensearchservice.OpenSearchService{}
	var err error
	if err = r.Client.Get(context.TODO(), request.NamespacedName, instance); err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

	r.StatusUpdater = util.NewStatusUpdater(r.Client, instance)
	if err = r.updateConditions(NewCondition(statusFalse,
		typeInProgress,
		opensearchServiceConditionReason,
		"Reconciliation cycle started")); err != nil {
		return ctrl.Result{}, err
	}
	defer func() {
		var status opensearchservice.StatusCondition
		if err != nil {
			status = NewCondition(statusFalse,
				typeFailed,
				opensearchServiceConditionReason,
				fmt.Sprintf("Reconciliation cycle is failed: %s", err.Error()))
		} else {
			status = NewCondition(statusTrue,
				typeSuccessful,
				opensearchServiceConditionReason,
				"Reconciliation cycle is successfully finished")
		}
		err = r.updateConditions(status)
		if err != nil {
			reqLogger.Error(err, "Unable to update custom resource conditions")
		}
	}()

	opensearchSecretName := fmt.Sprintf("%s-secret", instance.Name)
	opensearchSecretHash, err = r.calculateSecretDataHash(opensearchSecretName, opensearchSecretHashName, instance, log)
	if err != nil {
		return ctrl.Result{}, err
	}

	reconcilers := r.buildReconcilers(instance, log)

	for _, reconciler := range reconcilers {
		if err = reconciler.Reconcile(); err != nil {
			reqLogger.Error(err, fmt.Sprintf("Error when reconciling `%T`", reconciler))
			return ctrl.Result{}, err
		}
	}

	if instance.Spec.OpenSearch != nil {
		var readinessTimeout time.Duration
		readinessTimeout, err = time.ParseDuration(instance.Spec.OpenSearch.ReadinessTimeout)
		if err != nil {
			log.Error(err, fmt.Sprintf("Readiness timeout is specified incorrectly, %s value is used",
				defaultReadinessTimeout))
			readinessTimeout = defaultReadinessTimeout
		}
		err = wait.Poll(time.Second*20, readinessTimeout, func() (bool, error) {
			err := r.checkOpenSearchIsReady(instance)
			if err != nil {
				log.Info(fmt.Sprintf("OpenSearch check - %v", err))
				return false, nil
			}
			return true, nil
		})
		if err != nil {
			err = fmt.Errorf("OpenSearch is not ready after %s", readinessTimeout)
			return ctrl.Result{}, err
		}
	}

	for _, reconciler := range reconcilers {
		if err = reconciler.Configure(); err != nil {
			reqLogger.Error(err, fmt.Sprintf("Reconciliation cycle failed for %T:", reconciler))
			return ctrl.Result{}, err
		}
	}

	reqLogger.Info("Reconciliation cycle succeeded")
	r.ResourceHashes[opensearchSecretHashName] = opensearchSecretHash
	return ctrl.Result{}, nil
}

func (r *OpenSearchServiceReconciler) buildReconcilers(cr *opensearchservice.OpenSearchService,
	logger logr.Logger) []ReconcileService {
	var reconcilers []ReconcileService
	if cr.Spec.OpenSearch != nil {
		reconcilers = append(reconcilers, NewOpenSearchReconciler(r, cr, logger))
	}
	if cr.Spec.DisasterRecovery != nil {
		reconcilers = append(reconcilers, NewDisasterRecoveryReconciler(r, cr, logger))
	}
	if cr.Spec.Dashboards != nil {
		reconcilers = append(reconcilers, NewDashboardsReconciler(r, cr, logger))
	}
	if cr.Spec.Monitoring != nil {
		reconcilers = append(reconcilers, NewMonitoringReconciler(r, cr, logger))
	}
	if cr.Spec.DbaasAdapter != nil {
		reconcilers = append(reconcilers, NewDbaasAdapterReconciler(r, cr, logger))
	}
	if cr.Spec.ElasticsearchDbaasAdapter != nil {
		reconcilers = append(reconcilers, NewElasticsearchDbaasAdapterReconciler(r, cr, logger))
	}
	if cr.Spec.Curator != nil {
		reconcilers = append(reconcilers, NewCuratorReconciler(r, cr, logger))
	}
	return reconcilers
}

func (r *OpenSearchServiceReconciler) checkOpenSearchIsReady(cr *opensearchservice.OpenSearchService) error {
	credentials := r.parseOpenSearchCredentials(cr, log)
	url := r.createUrl(cr.Name, opensearchHttpPort)
	httpClient, err := r.configureClient()
	if err != nil {
		return NotReadyError{Err: err}
	}
	restClient := util.NewRestClient(url, httpClient, credentials)
	statusCode, _, err := restClient.SendRequest(http.MethodGet, "", nil)
	if err != nil || statusCode != 200 {
		return NotReadyError{StatusCode: statusCode, Err: err}
	}
	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *OpenSearchServiceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	statusPredicate := predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Ignore updates to CR status in which case metadata.Generation does not change
			if value, ok := e.ObjectNew.GetAnnotations()[util.SwitchoverAnnotationKey]; ok {
				if value != e.ObjectOld.GetAnnotations()[util.SwitchoverAnnotationKey] {
					return true
				}
			}
			return e.ObjectNew.GetGeneration() == 0 || e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			// Evaluates to false if the object has been confirmed deleted.
			return !e.DeleteStateUnknown
		},
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&opensearchservice.OpenSearchService{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&appsv1.StatefulSet{}).
		WithEventFilter(statusPredicate).
		WithOptions(controller.Options{RateLimiter: customRateLimiter()}).
		Complete(r)
}

/*
RateLimiter is used for calculating a delay before the next operator's reconcile function call.
Use ExponentialFailureRateLimiter which increases the delay exponentially until the delay is greater than the maximum.
After that, the delay will be fixed and equal to the maximum delay (the maxDelay parameter).
*/
func customRateLimiter() workqueue.RateLimiter {
	maxDelay, _ := util.GetIntEnvironmentVariable("RECONCILE_PERIOD", maxRateLimiterDelay)
	return workqueue.NewItemExponentialFailureRateLimiter(minRateLimiterDelay*time.Second,
		time.Duration(maxDelay)*time.Second)
}
