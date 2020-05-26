package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElasticEnvProjectSpec defines the desired state of ElasticEnvProject
type ElasticEnvProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Image       string                        `json:"image,omitempty"`
	Resources   ElasticEnvProjectResources    `json:"resources,omitempty"`
	Environment []ElasticEnvProjectEnvVar     `json:"env,omitempty"`
	Ports       []ElasticEnvProjectPortMapper `json:"ports,omitempty"`
	HealthCheck ElasticEnvProjectHealthCheck  `json:"healthCheck,omitempty"`
	HostAlias   ElasticEnvProjectHostAlias    `json:"hostAlias,omitempty"`
	Path        []ElasticEnvProjectSubPath    `json:"path,omitempty"`
	Volumes     []ElasticEnvProjectVolume     `json:"volumes,omitempty"`
	Command     string                        `json:"command,omitempty"`
	Args        []string                      `json:"args,omitempty"`
}

// ElasticEnvProjectStatus defines the observed state of ElasticEnvProject
type ElasticEnvProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticEnvProject is the Schema for the elasticenvprojects API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=elasticenvprojects,scope=Namespaced
type ElasticEnvProject struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticEnvProjectSpec   `json:"spec,omitempty"`
	Status ElasticEnvProjectStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticEnvProjectList contains a list of ElasticEnvProject
type ElasticEnvProjectList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticEnvProject `json:"items"`
}

// ElasticEnvProjectResource 资源限制
type ElasticEnvProjectResource struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

type ElasticEnvProjectResources struct {
	Limits   ElasticEnvProjectResource `json:"limits,omitempty"`
	Requests ElasticEnvProjectResource `json:"requests,omitempty"`
}

type ElasticEnvProjectEnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}

type ElasticEnvProjectPortMapper struct {
	Protocol      string `json:"protocol,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Port          int    `json:"port"`
}

type ElasticEnvProjectHostAlias struct {
	Hostname string `json:"hostname"`
	IP       string `json:"ip"`
}

type ElasticEnvProjectHealthCheck struct {
	Path string `json:"path"`
	Port int    `json:"port"`
}

type ElasticEnvProjectSubPath struct {
	Path string `json:"path"`
	Host string `json:"host"`
}

type ElasticEnvProjectVolume struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type,omitempty"`
}

func init() {
	SchemeBuilder.Register(&ElasticEnvProject{}, &ElasticEnvProjectList{})
}
