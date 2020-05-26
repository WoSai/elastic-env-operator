package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElasticEnvMirrorSpec defines the desired state of ElasticEnvMirror
type ElasticEnvMirrorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// ElasticEnvMirrorStatus defines the observed state of ElasticEnvMirror
type ElasticEnvMirrorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticEnvMirror is the Schema for the elasticenvmirrors API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=elasticenvmirrors,scope=Namespaced
type ElasticEnvMirror struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticEnvMirrorSpec   `json:"spec,omitempty"`
	Status ElasticEnvMirrorStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticEnvMirrorList contains a list of ElasticEnvMirror
type ElasticEnvMirrorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticEnvMirror `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ElasticEnvMirror{}, &ElasticEnvMirrorList{})
}
