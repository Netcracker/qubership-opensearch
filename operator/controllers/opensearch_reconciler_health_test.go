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

//go:build unit

package controllers

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	opensearchservice "github.com/Netcracker/qubership-opensearch/operator/api/v1"
	"github.com/Netcracker/qubership-opensearch/operator/util"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const allNodesJoined = `[{"name":"opensearch-0"},{"name":"opensearch-1"},{"name":"opensearch-2"}]`

func newHealthTestStatefulSet(name string, replicas int32) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns"},
		Spec:       appsv1.StatefulSetSpec{Replicas: &replicas},
	}
}

func newHealthTestReconciler(statefulSetNames string, statefulSets ...*appsv1.StatefulSet) OpenSearchReconciler {
	builder := fake.NewClientBuilder()
	for _, sts := range statefulSets {
		builder = builder.WithObjects(sts)
	}
	return OpenSearchReconciler{
		logger: logr.Discard(),
		cr: &opensearchservice.OpenSearchService{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns"},
			Spec: opensearchservice.OpenSearchServiceSpec{
				OpenSearch: &opensearchservice.OpenSearch{StatefulSetNames: statefulSetNames},
			},
		},
		reconciler: &OpenSearchServiceReconciler{Client: builder.Build()},
	}
}

func newHealthTestClient(t *testing.T, health, nodes string) *util.RestClient {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + clusterHealthPath:
			_, _ = w.Write([]byte(health))
		case "/_cat/nodes":
			_, _ = w.Write([]byte(nodes))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return util.NewRestClient(server.URL, http.Client{}, util.Credentials{})
}

func TestIsOpenSearchHealthy(t *testing.T) {
	tests := []struct {
		name    string
		health  string
		nodes   string
		healthy bool
	}{
		{"green and all nodes joined", `{"status":"green"}`, allNodesJoined, true},
		// Reported scenario: auto-expanded system indices keep the cluster green on two of three nodes.
		{"green but node missing", `{"status":"green"}`, `[{"name":"opensearch-0"},{"name":"opensearch-1"}]`, false},
		{"yellow", `{"status":"yellow"}`, allNodesJoined, false},
	}
	r := newHealthTestReconciler("opensearch", newHealthTestStatefulSet("opensearch", 3))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthy, err := r.isOpenSearchHealthy(newHealthTestClient(t, tt.health, tt.nodes))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if healthy != tt.healthy {
				t.Errorf("expected healthy=%t, got %t", tt.healthy, healthy)
			}
		})
	}
}

func TestGetOpenSearchNodes(t *testing.T) {
	r := newHealthTestReconciler("opensearch-data,opensearch-master",
		newHealthTestStatefulSet("opensearch-data", 2), newHealthTestStatefulSet("opensearch-master", 1))
	names, err := r.getOpenSearchNodes()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"opensearch-data-0", "opensearch-data-1", "opensearch-master-0"}
	if !reflect.DeepEqual(names, expected) {
		t.Errorf("expected %v, got %v", expected, names)
	}
}
