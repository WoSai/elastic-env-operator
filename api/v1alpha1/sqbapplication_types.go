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
	"github.com/wosai/elastic-env-operator/domain/util"
	corev1 "k8s.io/api/core/v1"
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
	// +kubebuilder:default:=80
	ServicePort int `json:"servicePort"`
}

type ServiceSpec struct {
	Ports []corev1.ServicePort `json:"ports"`
}

type DeploySpec struct {
	Replicas     *int32                       `json:"replicas,omitempty"`
	Image        string                       `json:"image,omitempty"`
	Command      []string                     `json:"command,omitempty"`
	Args         []string                     `json:"args,omitempty"`
	HostAlias    []corev1.HostAlias           `json:"hostAliases,omitempty"`
	Resources    *corev1.ResourceRequirements `json:"resources,omitempty"`
	Env          []corev1.EnvVar              `json:"env,omitempty"`
	HealthCheck  *corev1.Probe                `json:"healthCheck,omitempty"`
	Volumes      []corev1.Volume              `json:"volumes,omitempty"`
	VolumeMounts []corev1.VolumeMount         `json:"volumeMounts,omitempty"`
	NodeAffinity []NodeAffinity               `json:"nodeAffinity,omitempty"`
	Lifecycle    *Lifecycle                   `json:"lifecycle,omitempty"`
}

type NodeAffinity struct {
	// +kubebuilder:default:=100
	Weight                         int32 `json:"weight"`
	corev1.NodeSelectorRequirement `json:",inline"`
}

type Lifecycle struct {
	Init             *InitHandler `json:"init,omitempty"`
	corev1.Lifecycle `json:",inline"`
}

type InitHandler struct {
	Exec *corev1.ExecAction `json:"exec"`
}

// SQBApplicationStatus defines the observed state of SQBApplication
type SQBApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Planes      map[string]int `json:"planes,omitempty"`
	Mirrors     map[string]int `json:"mirrors,omitempty"`
	Initialized bool           `json:"initialized,omitempty"`
	ErrorInfo   string         `json:"errorInfo,omitempty"`
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

//merge list和map都合并去重
func (sqbapplication *SQBApplication) Merge(oldSqbapplication *SQBApplication) {
	// annotation、label
	sqbapplication.Annotations = util.MergeStringMap(oldSqbapplication.Annotations, sqbapplication.Annotations)
	sqbapplication.Labels = util.MergeStringMap(oldSqbapplication.Labels, sqbapplication.Labels)
	// host
	hosts := append(oldSqbapplication.Spec.Hosts, sqbapplication.Spec.Hosts...)
	hostsMap := make(map[string]struct{})
	for _, host := range hosts {
		hostsMap[host] = struct{}{}
	}
	hosts = make([]string, 0)
	for host := range hostsMap {
		hosts = append(hosts, host)
	}
	sqbapplication.Spec.Hosts = hosts
	// subpath根据path去重
	subpaths := append(oldSqbapplication.Spec.Subpaths, sqbapplication.Spec.Subpaths...)
	subpathMap := make(map[string]Subpath)
	for _, subpath := range subpaths {
		subpathMap[subpath.Path] = subpath
	}
	subpaths = make([]Subpath, 0)
	for _, subpath := range subpathMap {
		subpaths = append(subpaths, subpath)
	}
	sqbapplication.Spec.Subpaths = subpaths
	// ports根据port去重
	ports := append(oldSqbapplication.Spec.Ports, sqbapplication.Spec.Ports...)
	portsMap := make(map[int32]corev1.ServicePort)
	for _, port := range ports {
		portsMap[port.Port] = port
	}
	ports = make([]corev1.ServicePort, 0)
	for _, port := range portsMap {
		ports = append(ports, port)
	}
	sqbapplication.Spec.Ports = ports
	// deploy去重
	sqbapplication.Spec.DeploySpec.Merge(&oldSqbapplication.Spec.DeploySpec)
}

func (deploy *DeploySpec) Merge(oldDeploy *DeploySpec) {

}

func init() {
	SchemeBuilder.Register(&SQBApplication{}, &SQBApplicationList{})
}
