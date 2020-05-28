package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElasticEnvPlaneSpec defines the desired state of ElasticEnvPlane
type ElasticEnvPlaneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Purpose ElasticEnvPlanePurpose `json:"purpose"`
	Owner   string                 `json:"owner,omitempty"`
	Comment string                 `json:"comment,omitempty"`
}

// ElasticEnvPlaneStatus defines the observed state of ElasticEnvPlane
type ElasticEnvPlaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Conditions status.Conditions    `json:"conditions"`
	Phase      ElasticEnvPlanePhase `json:"phase"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticEnvPlane is the Schema for the elasticenvplanes API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=elasticenvplanes,scope=Namespaced
type ElasticEnvPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ElasticEnvPlaneSpec   `json:"spec,omitempty"`
	Status ElasticEnvPlaneStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ElasticEnvPlaneList contains a list of ElasticEnvPlane
type ElasticEnvPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ElasticEnvPlane `json:"items"`
}

// ElasticEnvPlanePurpose Plane用途，分别对应开发、测试、基准环境
// +kubebuilder:validation:Enum=development;test;base
// +kubebuilder:default:=base
type ElasticEnvPlanePurpose string

// ElasticEnvPlanePhase Plane当前阶段
// +kubebuilder:validation:Enum=creating;ready;locked;deleting
// +kubebuilder:default:=creating
type ElasticEnvPlanePhase string

func init() {
	SchemeBuilder.Register(&ElasticEnvPlane{}, &ElasticEnvPlaneList{})
}
