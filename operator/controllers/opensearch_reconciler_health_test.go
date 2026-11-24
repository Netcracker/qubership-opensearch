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

func newHealthTestReconciler(spec opensearchservice.OpenSearch, statefulSets ...*appsv1.StatefulSet) OpenSearchReconciler {
	builder := fake.NewClientBuilder()
	for _, sts := range statefulSets {
		builder = builder.WithObjects(sts)
	}
	return OpenSearchReconciler{
		logger: logr.Discard(),
		cr: &opensearchservice.OpenSearchService{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns"},
			Spec:       opensearchservice.OpenSearchServiceSpec{OpenSearch: &spec},
		},
		reconciler: &OpenSearchServiceReconciler{Client: builder.Build()},
	}
}

// newHealthTestClient emulates OpenSearch API; empty nodes means the cluster manager is not discovered.
func newHealthTestClient(t *testing.T, health, nodes string) *util.RestClient {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/" + clusterHealthPath:
			_, _ = w.Write([]byte(health))
		case "/_cat/nodes":
			if nodes == "" {
				w.WriteHeader(http.StatusServiceUnavailable)
				return
			}
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
		wantErr bool
	}{
		{"green and all nodes joined", `{"status":"green"}`, allNodesJoined, true, false},
		// Reported scenario: auto-expanded system indices keep the cluster green on two of three nodes.
		{"green but node missing", `{"status":"green"}`, `[{"name":"opensearch-0"},{"name":"opensearch-1"}]`, false, false},
		{"yellow and all nodes joined", `{"status":"yellow"}`, allNodesJoined, false, false},
		{"cluster manager not discovered", `{"status":"green"}`, "", false, true},
	}
	r := newHealthTestReconciler(opensearchservice.OpenSearch{StatefulSetNames: "opensearch"},
		newHealthTestStatefulSet("opensearch", 3))
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			healthy, err := r.isOpenSearchHealthy(newHealthTestClient(t, tt.health, tt.nodes))
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%t, got %v", tt.wantErr, err)
			}
			if healthy != tt.healthy {
				t.Errorf("expected healthy=%t, got %t", tt.healthy, healthy)
			}
		})
	}
}

func TestGetOpenSearchNodes(t *testing.T) {
	statefulSets := []*appsv1.StatefulSet{
		newHealthTestStatefulSet("opensearch-data", 2), newHealthTestStatefulSet("opensearch-master", 1)}
	tests := []struct {
		name     string
		spec     opensearchservice.OpenSearch
		expected []string
		wantErr  bool
	}{
		{"all stateful sets", opensearchservice.OpenSearch{StatefulSetNames: "opensearch-data,opensearch-master"},
			[]string{"opensearch-data-0", "opensearch-data-1", "opensearch-master-0"}, false},
		// CR rendered by previous chart versions has no stateful set names when rolling update is disabled.
		{"master stateful set fallback", opensearchservice.OpenSearch{MasterStsName: "opensearch-master"},
			[]string{"opensearch-master-0"}, false},
		{"no stateful sets", opensearchservice.OpenSearch{}, nil, false},
		{"stateful set not found", opensearchservice.OpenSearch{StatefulSetNames: "opensearch-arbiter"}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			names, err := newHealthTestReconciler(tt.spec, statefulSets...).getOpenSearchNodes()
			if (err != nil) != tt.wantErr {
				t.Fatalf("expected error=%t, got %v", tt.wantErr, err)
			}
			if !reflect.DeepEqual(names, tt.expected) {
				t.Errorf("expected %v, got %v", tt.expected, names)
			}
		})
	}
}
