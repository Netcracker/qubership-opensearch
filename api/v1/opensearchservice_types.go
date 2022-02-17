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
}

// Dashboards structure defines parameters necessary for interaction with Dashboards
type Dashboards struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName,omitempty"`
}

// Monitoring structure defines parameters necessary for interaction with OpenSearch monitoring
type Monitoring struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName,omitempty"`
}

// DbaasAdapter structure defines parameters necessary for interaction with DBaaS OpenSearch adapter
type DbaasAdapter struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName"`
}

// Curator structure defines parameters necessary for interaction with OpenSearch Curator
type Curator struct {
	Name       string `json:"name"`
	SecretName string `json:"secretName"`
}

// DisasterRecovery shows Disaster Recovery configuration
type DisasterRecovery struct {
	Mode          string `json:"mode"`
	NoWait        bool   `json:"noWait,omitempty"`
	ConfigMapName string `json:"configMapName"`
}

// OpenSearchServiceSpec defines the desired state of OpenSearchService
type OpenSearchServiceSpec struct {
	// Important: Run "make" to regenerate code after modifying this file
	OpenSearch       *OpenSearch       `json:"opensearch,omitempty"`
	Dashboards       *Dashboards       `json:"dashboards,omitempty"`
	Monitoring       *Monitoring       `json:"monitoring,omitempty"`
	DbaasAdapter     *DbaasAdapter     `json:"dbaasAdapter,omitempty"`
	Curator          *Curator          `json:"curator,omitempty"`
	DisasterRecovery *DisasterRecovery `json:"disasterRecovery,omitempty"`
}

type DisasterRecoveryStatus struct {
	Mode    string `json:"mode"`
	Status  string `json:"status"`
	Comment string `json:"comment,omitempty"`
}

// OpenSearchServiceStatus defines the observed state of OpenSearchService
type OpenSearchServiceStatus struct {
	// Important: Run "make" to regenerate code after modifying this file
	DisasterRecoveryStatus DisasterRecoveryStatus `json:"disasterRecoveryStatus,omitempty"`
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
