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

func init() {
	SchemeBuilder.Register(&ElasticEnvProject{}, &ElasticEnvProjectList{})
}
