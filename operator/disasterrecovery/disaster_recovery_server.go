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

package disasterrecovery

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	OK                  = 200
	InternalServerError = 500
	DEGRADED            = "degraded"
	DOWN                = "down"
	UP                  = "up"
	replicationName     = "dr-replication"
)

var (
	verbose = GetEnv("DEBUG", "false")
	log     = logf.Log.WithName("dr_health_server")
)

type ServerContext struct {
	replicationChecker ReplicationChecker
}

type ClusterState struct {
	Status string `json:"status"`
}

func StartServer(replicationChecker ReplicationChecker) error {
	serverContext := ServerContext{replicationChecker: replicationChecker}
	server := &http.Server{
		Addr:    ":8069",
		Handler: ServerHandlers(serverContext),
	}
	return server.ListenAndServe()
}

func ServerHandlers(serverContext ServerContext) http.Handler {
	r := mux.NewRouter()
	r.Handle("/healthz", http.HandlerFunc(serverContext.GetClusterHealthStatus())).Methods("GET")
	return JsonContentType(handlers.CompressHandler(r))
}

func (serverContext ServerContext) GetClusterHealthStatus() func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		mode, ok := r.URL.Query()["mode"]
		if !ok {
			log.Error(fmt.Errorf("parameter mode is not presented in the url"),
				"Health endpoint expects mode http parameter")
			sendFailedHealthResponse(w)
			return
		}
		if mode[0] == "active" || mode[0] == "disable" {
			sendSuccessfulResponse(w, ClusterState{Status: UP})
			return
		}
		if mode[0] != "standby" {
			log.Error(fmt.Errorf("unexpected mode http parameter"),
				fmt.Sprintf("mode parameter must be in the list of values [active, standby, disable]. But %s is given",
					mode[0]))
			sendFailedHealthResponse(w)
			return
		}
		status, err := serverContext.replicationChecker.CheckReplication()
		if err != nil {
			sendFailedHealthResponse(w)
			return
		}

		clusterState := ClusterState{Status: status}
		sendSuccessfulResponse(w, clusterState)
	}
}

func sendFailedHealthResponse(w http.ResponseWriter) {
	response := ClusterState{
		Status: DOWN,
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
