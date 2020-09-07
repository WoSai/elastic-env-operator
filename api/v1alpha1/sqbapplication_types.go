/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SQBApplicationSpec defines the desired state of SQBApplication
type SQBApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	IngressSpec `json:",inline"`
	ServiceSpec `json:",inline"`
	DeploySpec  `json:",inline"`
}

type IngressSpec struct {
	Hosts    []string  `json:"hosts,omitempty"`
	Subpaths []Subpath `json:"subpaths,omitempty"`
}

type Subpath struct {
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	ServicePort int    `json:"servicePort"`
}

type ServiceSpec struct {
	ServiceType v1.ServiceType   `json:"serviceType,omitempty"`
	Ports       []v1.ServicePort `json:"ports"`
}

type DeploySpec struct {
	Replicas     *int32                   `json:"replicas,omitempty"`
	Image        string                   `json:"image,omitempty"`
	Resources    *v1.ResourceRequirements `json:"resources,omitempty"`
	Env          []v1.EnvVar              `json:"env,omitempty"`
	EnvFrom      []v1.EnvFromSource       `json:"envFrom,omitempty"`
	HealthCheck  *v1.Probe                `json:"healthCheck,omitempty"`
	Volumes      []v1.Volume              `json:"volumes,omitempty"`
	VolumeMounts []v1.VolumeMount         `json:"volumeMounts,omitempty"`
	NodeAffinity []NodeAffinity           `json:"nodeAffinity,omitempty"`
	Lifecycle    *Lifecycle               `json:"lifecycle,omitempty"`
}

type NodeAffinity struct {
	Weight *int32   `json:"weight,omitempty"`
	Key    string   `json:"key"`
	Values []string `json:"values"`
}

type Lifecycle struct {
	Init         *InitHandler `json:"init,omitempty"`
	v1.Lifecycle `json:",inline"`
}

type InitHandler struct {
	Exec *v1.ExecAction `json:"exec"`
}

// SQBApplicationStatus defines the observed state of SQBApplication
type SQBApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Planes      map[string]string `json:"planes,omitempty"`
	Mirrors     map[string]string `json:"mirrors,omitempty"`
	Initialized bool              `json:"initialized,omitempty"`
	ErrorInfo   string            `json:"errorInfo,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SQBApplication is the Schema for the sqbapplications API
type SQBApplication struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SQBApplicationSpec   `json:"spec,omitempty"`
	Status SQBApplicationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SQBApplicationList contains a list of SQBApplication
type SQBApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQBApplication `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SQBApplication{}, &SQBApplicationList{})
}
