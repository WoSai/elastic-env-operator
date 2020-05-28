package v1alpha1

import (
	"github.com/operator-framework/operator-sdk/pkg/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ElasticEnvProjectSpec defines the desired state of ElasticEnvProject
// +k8s:openapi-gen=true
type ElasticEnvProjectSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Image string `json:"image,omitempty"`
	// +kubebuilder:default:=1
	Replicas     int32                         `json:"replicas,omitempty"`
	NodeSelector map[string]string             `json:"nodeSelector,omitempty"`
	Resources    corev1.ResourceRequirements   `json:"resources,omitempty"`
	EnvVars      []corev1.EnvVar               `json:"env,omitempty"`
	Ports        []ElasticEnvProjectPortMapper `json:"ports,omitempty"`
	HealthCheck  ElasticEnvProjectHealthCheck  `json:"healthCheck,omitempty"`
	HostAlias    []corev1.HostAlias            `json:"hostAlias,omitempty"`
	Path         []ElasticEnvProjectSubPath    `json:"path,omitempty"`
	Volumes      []ElasticEnvProjectVolume     `json:"volumes,omitempty"`
	Command      []string                      `json:"command,omitempty"`
	Args         []string                      `json:"args,omitempty"`
	// DisableIstio 当为true时，将不再创建Istio下的Gateway/VirtualService/DestinationRule等对象，只使用K8s原生对象
	// +kubebuilder:default:=false
	DisableIstio bool `json:"diableIstio,omitempty"`
}

// ElasticEnvProjectStatus defines the observed state of ElasticEnvProject
// +k8s:openapi-gen=true
type ElasticEnvProjectStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Conditions status.Conditions `json:"condition"`
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

// ElasticEnvProjectPortMapper 对口映射规则
type ElasticEnvProjectPortMapper struct {
	// Protocol istio支持的协议列表，见：https://istio.io/docs/ops/configuration/traffic-management/protocol-selection/
	// +kubebuilder:default:=http
	// +kubebuilder:validation:Enum:=grpc;grpc-web;http;http2;https;mongo;mysql;redis;tcp;tls;udp
	Protocol      string `json:"protocol,omitempty"`
	ContainerPort int    `json:"containerPort"`
	Port          int    `json:"port"`
}

// ElasticEnvProjectHealthCheck 健康检查配置
type ElasticEnvProjectHealthCheck struct {
	// +kubebuilder:default:=/
	Path string `json:"path"`
	Port int    `json:"port"`
}

// ElasticEnvProjectSubPath Ingress子路径映射配置
type ElasticEnvProjectSubPath struct {
	Path string `json:"path"`
	Host string `json:"host"`
}

// ElasticEnvProjectVolume volume映射规则
type ElasticEnvProjectVolume struct {
	Source    corev1.VolumeSource `json:"source"`
	MountPath string              `json:"mountPath"`
}

func init() {
	SchemeBuilder.Register(&ElasticEnvProject{}, &ElasticEnvProjectList{})
}
