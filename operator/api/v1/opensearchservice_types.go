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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OpenSearch structure defines parameters necessary for interaction with OpenSearch
type OpenSearch struct {
	DedicatedClientPod        bool       `json:"dedicatedClientPod"`
	DedicatedDataPod          bool       `json:"dedicatedDataPod"`
	Snapshots                 *Snapshots `json:"snapshots,omitempty"`
	SecurityConfigurationName string     `json:"securityConfigurationName"`
	CompatibilityModeEnabled  bool       `json:"compatibilityModeEnabled,omitempty"`
	RollingUpdate             bool       `json:"rollingUpdate,omitempty"`
	StatefulSetNames          string     `json:"statefulSetNames,omitempty"`
	ReadinessTimeout          string     `json:"readinessTimeout,omitempty"`
	DisabledRestCategories    []string   `json:"disabledRestCategories,omitempty"`
}

type ExternalOpenSearch struct {
	Config map[string]string `json:"config"`
	Url    string            `json:"url"`
}

type Snapshots struct {
	RepositoryName string `json:"repositoryName"`
	S3             *S3    `json:"s3,omitempty"`
}

type S3 struct {
	Enabled         bool   `json:"enabled,omitempty"`
	PathStyleAccess bool   `json:"pathStyleAccess,omitempty"`
	Url             string `json:"url,omitempty"`
	Bucket          string `json:"bucket,omitempty"`
	BasePath        string `json:"basePath,omitempty"`
	Region          string `json:"region,omitempty"`
	SecretName      string `json:"secretName,omitempty"`
	GcsEnabled      bool   `json:"gcsEnabled,omitempty"`
}

// Dashboards structure defines parameters necessary for interaction with Dashboards
type Dashboards struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName,omitempty"`
}

// Monitoring structure defines parameters necessary for interaction with OpenSearch monitoring
type Monitoring struct {
	Name        string       `json:"name"`
	SecretName  string       `json:"secretName,omitempty"`
	SlowQueries *SlowQueries `json:"slowQueries,omitempty"`
}

type SlowQueries struct {
	IndicesPattern string `json:"indicesPattern"`
	MinSeconds     int    `json:"minSeconds"`
}

// DbaasAdapter structure defines parameters necessary for interaction with DBaaS OpenSearch adapter
type DbaasAdapter struct {
	AdapterAddress             string `json:"adapterAddress,omitempty"`
	AggregatorAddress          string `json:"aggregatorAddress,omitempty"`
	Name                       string `json:"name"`
	PhysicalDatabaseIdentifier string `json:"physicalDatabaseIdentifier,omitempty"`
	SecretName                 string `json:"secretName"`
}

// ElasticsearchDbaasAdapter structure defines parameters necessary for interaction with DBaaS Elasticsearch adapter
type ElasticsearchDbaasAdapter struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName,omitempty"`
}

// Curator structure defines parameters necessary for interaction with OpenSearch Curator
type Curator struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName"`
}

// DisasterRecovery shows Disaster Recovery configuration
type DisasterRecovery struct {
	Mode                       string `json:"mode"`
	NoWait                     bool   `json:"noWait,omitempty"`
	ConfigMapName              string `json:"configMapName"`
	ReplicationWatcherEnabled  bool   `json:"replicationWatcherEnabled,omitempty"`
	ReplicationWatcherInterval int    `json:"replicationWatcherInterval,omitempty"`
}

// OpenSearchServiceSpec defines the desired state of OpenSearchService
type OpenSearchServiceSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	OpenSearch                *OpenSearch                `json:"opensearch,omitempty"`
	ExternalOpenSearch        *ExternalOpenSearch        `json:"externalOpenSearch,omitempty"`
	Dashboards                *Dashboards                `json:"dashboards,omitempty"`
	Monitoring                *Monitoring                `json:"monitoring,omitempty"`
	DbaasAdapter              *DbaasAdapter              `json:"dbaasAdapter,omitempty"`
	ElasticsearchDbaasAdapter *ElasticsearchDbaasAdapter `json:"elasticsearchDbaasAdapter,omitempty"`
	Curator                   *Curator                   `json:"curator,omitempty"`
	DisasterRecovery          *DisasterRecovery          `json:"disasterRecovery,omitempty"`
}

type DisasterRecoveryStatus struct {
	Mode               string `json:"mode"`
	Status             string `json:"status"`
	Comment            string `json:"comment,omitempty"` // deprecated
	Message            string `json:"message,omitempty"`
	UsersRecoveryState string `json:"usersRecoveryState,omitempty"`
}

// OpenSearchServiceStatus defines the observed state of OpenSearchService
type OpenSearchServiceStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	DisasterRecoveryStatus DisasterRecoveryStatus `json:"disasterRecoveryStatus,omitempty"`
	Conditions             []StatusCondition      `json:"conditions,omitempty"`
	RollingUpdateStatus    RollingUpdateStatus    `json:"rollingUpdateStatus,omitempty"`
}

type RollingUpdateStatus struct {
	Status              string              `json:"status,omitempty"`
	StatefulSetStatuses []StatefulSetStatus `json:"statefulSetStatuses,omitempty"`
}

type StatefulSetStatus struct {
	Name                      string  `json:"name,omitempty"`
	LastStatefulSetGeneration int64   `json:"lastStatefulSetGeneration,omitempty"`
	UpdatedReplicas           []int32 `json:"updatedReplicas,omitempty"`
}

// StatusCondition contains description of status of OpenSearchService
// +k8s:openapi-gen=true
type StatusCondition struct {
	// Type - Can be "In progress", "Failed", "Successful" or "Ready".
	Type string `json:"type"`
	// Status - "True" if condition is successfully done and "False" if condition has failed or in progress type.
	Status string `json:"status"`
	// Reason - One-word CamelCase reason for the condition's state.
	Reason string `json:"reason"`
	// Message - Human-readable message indicating details about last transition.
	Message string `json:"message"`
	// LastTransitionTime - Last time the condition transit from one status to another.
	LastTransitionTime string `json:"lastTransitionTime"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:storageversion

// OpenSearchService is the Schema for the opensearchservices API
type OpenSearchService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   OpenSearchServiceSpec   `json:"spec,omitempty"`
	Status OpenSearchServiceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// OpenSearchServiceList contains a list of OpenSearchService
type OpenSearchServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OpenSearchService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OpenSearchService{}, &OpenSearchServiceList{})
}
