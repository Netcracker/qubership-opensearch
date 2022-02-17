package disasterrecovery

import (
	"context"
	"encoding/json"
	"fmt"
	opensearchservice "git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/api/v1"
	"git.netcracker.com/PROD.Platform.ElasticStack/opensearch-service/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"io/ioutil"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"net/http"
	"os"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"time"
)

const (
	ACTIVE              = "active"
	DISABLED            = "disable"
	STANDBY             = "standby"
	FAILED              = "failed"
	RUNNING             = "running"
	DONE                = "done"
	OK                  = 200
	InternalServerError = 500
	DEGRADED            = "degraded"
	DOWN                = "down"
	UP                  = "up"
)

var (
	verbose   = GetEnv("DEBUG", "false")
	namespace = GetEnv("OPERATOR_NAMESPACE", "")
)

type ServerContext struct {
	client client.Client
}

type RequestData struct {
	Mode   string `json:"mode"`
	NoWait *bool  `json:"no-wait,omitempty"`
}

type ClusterState struct {
	Status string `json:"status"`
}

type SwitchoverState struct {
	Mode    string `json:"mode"`
	Status  string `json:"status,omitempty"`
	Comment string `json:"comment,omitempty"`
}

func StartServer(client client.Client) error {
	serverContext := ServerContext{client: client}
	server := &http.Server{
		Addr:    ":8068",
		Handler: ServerHandlers(serverContext),
	}
	return server.ListenAndServe()
}

func ServerHandlers(serverContext ServerContext) http.Handler {
	r := mux.NewRouter()
	r.Handle("/healthz", http.HandlerFunc(serverContext.GetClusterHealthStatus())).Methods("GET")
	r.Handle("/sitemanager", http.HandlerFunc(serverContext.GetModeAndStatus())).Methods("GET")
	r.Handle("/sitemanager", http.HandlerFunc(serverContext.SetMode())).Methods("POST")
	return JsonContentType(handlers.CompressHandler(r))
}

func (serverContext ServerContext) GetClusterHealthStatus() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		customResource, err := serverContext.getCustomResource()
		if err != nil || customResource == nil {
			sendFailedHealthResponse(w)
			return
		}

		opensearchStatus, err := serverContext.getServiceStatus(GetOpenSearchLabels(customResource.Name))
		if err != nil {
			sendFailedHealthResponse(w)
			return
		}

		//TODO: check replication establish instead of KMM
		clusterState := ClusterState{Status: opensearchStatus}
		sendSuccessfulResponse(w, clusterState)
	}
}

func (serverContext ServerContext) GetModeAndStatus() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		customResource, err := serverContext.getCustomResource()
		if err != nil || customResource == nil {
			sendFailedSwitchoverResponse(w, "",
				fmt.Sprintf("Custom resource OpenSearchservice is not found in the namespace - %s. Error is [%v]", namespace, err))
		} else {
			response := SwitchoverState{
				Mode:    customResource.Status.DisasterRecoveryStatus.Mode,
				Status:  customResource.Status.DisasterRecoveryStatus.Status,
				Comment: customResource.Status.DisasterRecoveryStatus.Comment,
			}
			sendSuccessfulResponse(w, response)
		}
	}
}

func (serverContext ServerContext) SetMode() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var data RequestData
		requestBody, err := ioutil.ReadAll(r.Body)
		err = json.Unmarshal(requestBody, &data)
		if err != nil {
			sendFailedSwitchoverResponse(w, "",
				fmt.Sprintf("Unmarshalling data from request failed. Error is [%v]", err))
			return
		}

		mode := data.Mode
		if mode != ACTIVE && mode != STANDBY && mode != DISABLED {
			sendFailedSwitchoverResponse(w, mode,
				fmt.Sprintf("'%s' mode is not in the allowed list. Please, use '%s', '%s' or '%s'", mode, ACTIVE, STANDBY, DISABLED))
			return
		}

		customResource, err := serverContext.getCustomResource()
		if err != nil || customResource == nil {
			sendFailedSwitchoverResponse(w, mode,
				fmt.Sprintf("Custom resource OpenSearchService is not found in the namespace - %s. Error is [%v]", namespace, err))
			return
		}

		if customResource.Status.DisasterRecoveryStatus.Status == RUNNING {
			sendFailedSwitchoverResponse(w, mode,
				"Switchover process in progress. Please, wait until it will be finished")
			return
		}

		if customResource.Status.DisasterRecoveryStatus.Mode == customResource.Spec.DisasterRecovery.Mode &&
			customResource.Status.DisasterRecoveryStatus.Mode == mode &&
			customResource.Status.DisasterRecoveryStatus.Status == DONE {
			sendFailedSwitchoverResponse(w, mode,
				"Switchover process has already done")
			return
		}

		var noWait bool
		if data.NoWait == nil {
			noWait = true
		} else {
			noWait = *data.NoWait
		}

		if customResource.Spec.DisasterRecovery != nil {
			customResource.Spec.DisasterRecovery.Mode = mode
			customResource.Spec.DisasterRecovery.NoWait = noWait
			// we should send CR to reconcile loop if switchover from standby to active was failed
			// and we get the same request again
			if isRetryAction(customResource, mode) {
				annotations := customResource.GetAnnotations()
				annotations[util.SwitchoverAnnotationKey] = strconv.Itoa(time.Now().Nanosecond())
				customResource.SetAnnotations(annotations)
			}
		} else {
			customResource.Spec.DisasterRecovery = &opensearchservice.DisasterRecovery{Mode: mode, NoWait: noWait}
		}

		_, err = serverContext.updateCustomResource(customResource)
		if err != nil {
			sendFailedSwitchoverResponse(w, mode, err.Error())
			return
		}

		statusUpdater := util.NewStatusUpdater(serverContext.client, customResource)
		err = statusUpdater.UpdateStatusWithRetry(func(instance *opensearchservice.OpenSearchService) {
			setStatus(instance, customResource)
		})
		if err != nil {
			sendFailedSwitchoverResponse(w, mode, err.Error())
			return
		}

		sendSuccessfulResponse(w, SwitchoverState{Mode: mode})
	}
}

func (serverContext ServerContext) getCustomResource() (*opensearchservice.OpenSearchService, error) {
	result := &opensearchservice.OpenSearchServiceList{}
	err := serverContext.client.List(context.TODO(), result, &client.ListOptions{Namespace: namespace})
	if len(result.Items) == 0 {
		return nil, err
	}
	return &result.Items[0], err
}

func (serverContext ServerContext) updateCustomResource(customResource *opensearchservice.OpenSearchService) (*opensearchservice.OpenSearchService, error) {
	result := &opensearchservice.OpenSearchService{}
	err := serverContext.client.Update(context.TODO(), customResource)
	if err == nil {
		result, err = serverContext.getCustomResource()
	}
	return result, err
}

func setStatus(instance *opensearchservice.OpenSearchService, currentCr *opensearchservice.OpenSearchService) {
	if currentCr.Status.DisasterRecoveryStatus.Mode != currentCr.Spec.DisasterRecovery.Mode {
		instance.Status.DisasterRecoveryStatus.Status = RUNNING
		instance.Status.DisasterRecoveryStatus.Comment = "The request is accepted"
	} else {
		if currentCr.Status.DisasterRecoveryStatus.Status == FAILED {
			instance.Status.DisasterRecoveryStatus.Comment = util.RetryFailedComment
		}
	}
}

func GetOpenSearchLabels(customResourceName string) map[string]string {
	return map[string]string{
		"app": customResourceName,
	}
}

func (serverContext ServerContext) getServiceStatus(deploymentLabels map[string]string) (status string, err error) {
	readyStatefulSetsNumber, allStatefulSetsNumber, err := serverContext.getStatefulSetData(deploymentLabels)
	if err != nil {
		return
	}
	if readyStatefulSetsNumber == 0 {
		status = DOWN
	} else if readyStatefulSetsNumber < allStatefulSetsNumber {
		status = DEGRADED
	} else {
		status = UP
	}
	return
}

func (serverContext ServerContext) getStatefulSetData(statefulSetLabels map[string]string) (readyStatefulSetsNumber, allStatefulSetsNumber int, err error) {
	statefulSets := &appsv1.StatefulSetList{}
	err = serverContext.client.List(context.TODO(), statefulSets, &client.ListOptions{
		Namespace:     namespace,
		LabelSelector: labels.SelectorFromSet(statefulSetLabels),
	})
	if err != nil {
		return
	}

	allStatefulSetsNumber = len(statefulSets.Items)
	for _, statefulSet := range statefulSets.Items {
		if IsStatefulSetReady(statefulSet) {
			readyStatefulSetsNumber += 1
		}
	}
	return
}

func IsStatefulSetReady(statefulSet appsv1.StatefulSet) bool {
	availableReplicas := util.Min(statefulSet.Status.ReadyReplicas, statefulSet.Status.UpdatedReplicas)
	return *statefulSet.Spec.Replicas == availableReplicas && *statefulSet.Spec.Replicas != 0
}

func sendFailedHealthResponse(w http.ResponseWriter) {
	response := ClusterState{
		Status: DOWN,
	}
	sendResponse(w, InternalServerError, response)
}

func sendFailedSwitchoverResponse(w http.ResponseWriter, mode string, comment string) {
	response := SwitchoverState{
		Mode:    mode,
		Status:  FAILED,
		Comment: comment,
	}
	sendResponse(w, InternalServerError, response)
}

func sendSuccessfulResponse(w http.ResponseWriter, response interface{}) {
	sendResponse(w, OK, response)
}

func sendResponse(w http.ResponseWriter, statusCode int, response interface{}) {
	w.WriteHeader(statusCode)
	responseBody, _ := json.Marshal(response)
	if verbose == "true" {
		fmt.Printf("Response body: %s\n", responseBody)
	}
	_, _ = w.Write(responseBody)
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func JsonContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		h.ServeHTTP(w, r)
	})
}

func isRetryAction(cr *opensearchservice.OpenSearchService, newMode string) bool {
	return newMode == cr.Spec.DisasterRecovery.Mode && cr.Status.DisasterRecoveryStatus.Status != "done"
}
