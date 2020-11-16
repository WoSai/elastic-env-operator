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
	"encoding/json"
	prometheus "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	jsonpatch "github.com/evanphx/json-patch"
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
	Domains  []Domain  `json:"domains,omitempty"`
	Subpaths []Subpath `json:"subpaths,omitempty"`
}

type Domain struct {
	Class      string `json:"class"`
	Annotation string `json:"annotation,omitempty"`
	Host       string `json:"host,omitempty"`
}

type Subpath struct {
	Path        string `json:"path"`
	ServiceName string `json:"serviceName"`
	// +kubebuilder:default:=80
	ServicePort int `json:"servicePort"`
}

type ServiceSpec struct {
	Ports    []corev1.ServicePort  `json:"ports"`
	Monitors []prometheus.Endpoint `json:"monitors,omitempty"`
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
	NodeAffinity *NodeAffinity                `json:"nodeAffinity,omitempty"`
	Lifecycle    *Lifecycle                   `json:"lifecycle,omitempty"`
}

type NodeAffinity struct {
	Require  []NodeSelector `json:"require,omitempty"`
	Prefer []NodeSelector `json:"prefer,omitempty"`
}

type NodeSelector struct {
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

	Planes    map[string]int `json:"planes,omitempty"`
	Mirrors   map[string]int `json:"mirrors,omitempty"`
	ErrorInfo string         `json:"errorInfo,omitempty"`
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
func (old *SQBApplication) Merge(news *SQBApplication) {
	// annotation、label
	old.Annotations = util.MergeStringMap(old.Annotations, news.Annotations)
	old.Labels = util.MergeStringMap(old.Labels, news.Labels)
	// domain使用新的覆盖
	old.Spec.Domains = news.Spec.Domains
	// subpath用新的覆盖
	old.Spec.Subpaths = news.Spec.Subpaths
	// ports用新的覆盖
	old.Spec.Ports = news.Spec.Ports
	// deploy去重
	old.Spec.DeploySpec.merge(&news.Spec.DeploySpec)
}

func (old *DeploySpec) merge(news *DeploySpec) {
	// 先做merge patch
	originOld := old.DeepCopy()
	deployByte, _ := json.Marshal(news)
	oldDeployByte, _ := json.Marshal(old)
	mergeDeployByte, _ := jsonpatch.MergePatch(oldDeployByte, deployByte)
	_ = json.Unmarshal(mergeDeployByte, &old)
	// hostalias根据ip去重
	hostaliases := append(originOld.HostAlias, old.HostAlias...)
	hostaliasMap := make(map[string]corev1.HostAlias)
	for _, hostalias := range hostaliases {
		hostaliasMap[hostalias.IP] = hostalias
	}
	hostaliases = make([]corev1.HostAlias, 0)
	for _, hostalias := range hostaliasMap {
		hostaliases = append(hostaliases, hostalias)
	}
	old.HostAlias = hostaliases
	// env根据name去重
	envs := append(originOld.Env, old.Env...)
	envMap := make(map[string]corev1.EnvVar)
	for _, env := range envs {
		envMap[env.Name] = env
	}
	envs = make([]corev1.EnvVar, 0)
	for _, env := range envMap {
		envs = append(envs, env)
	}
	old.Env = envs
	// volumes根据name去重
	volumes := append(originOld.Volumes, old.Volumes...)
	volumeMap := make(map[string]corev1.Volume)
	for _, volume := range volumes {
		volumeMap[volume.Name] = volume
	}
	volumes = make([]corev1.Volume, 0)
	for _, volume := range volumeMap {
		volumes = append(volumes, volume)
	}
	old.Volumes = volumes
	// volumeMounts根据name去重
	volumeMounts := append(originOld.VolumeMounts, old.VolumeMounts...)
	volumeMountsMap := make(map[string]corev1.VolumeMount)
	for _, volumeMount := range volumeMounts {
		volumeMountsMap[volumeMount.Name] = volumeMount
	}
	volumeMounts = make([]corev1.VolumeMount, 0)
	for _, volumeMount := range volumeMountsMap {
		volumeMounts = append(volumeMounts, volumeMount)
	}
	old.VolumeMounts = volumeMounts
}

func init() {
	SchemeBuilder.Register(&SQBApplication{}, &SQBApplicationList{})
}
