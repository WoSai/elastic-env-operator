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
	Initialized bool           `json:"initialized,omitempty"` // 废弃了，使用注解qa.shouqianba.com/initialized判断初始化
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
func (old *SQBApplication) Merge(new *SQBApplication) {
	// annotation、label
	old.Annotations = util.MergeStringMap(old.Annotations, new.Annotations)
	old.Labels = util.MergeStringMap(old.Labels, new.Labels)
	// host
	hosts := append(old.Spec.Hosts, new.Spec.Hosts...)
	hostsMap := make(map[string]struct{})
	for _, host := range hosts {
		hostsMap[host] = struct{}{}
	}
	hosts = make([]string, 0)
	for host := range hostsMap {
		hosts = append(hosts, host)
	}
	old.Spec.Hosts = hosts
	// subpath根据path去重
	subpaths := append(old.Spec.Subpaths, new.Spec.Subpaths...)
	subpathMap := make(map[string]Subpath)
	for _, subpath := range subpaths {
		subpathMap[subpath.Path] = subpath
	}
	subpaths = make([]Subpath, 0)
	for _, subpath := range subpathMap {
		subpaths = append(subpaths, subpath)
	}
	old.Spec.Subpaths = subpaths
	// ports根据port去重
	ports := append(old.Spec.Ports, new.Spec.Ports...)
	portsMap := make(map[int32]corev1.ServicePort)
	for _, port := range ports {
		portsMap[port.Port] = port
	}
	ports = make([]corev1.ServicePort, 0)
	for _, port := range portsMap {
		ports = append(ports, port)
	}
	old.Spec.Ports = ports
	// deploy去重
	old.Spec.DeploySpec.merge(&new.Spec.DeploySpec)
}

func (old *DeploySpec) merge(new *DeploySpec) {
	// 先做merge patch
	originOld := old.DeepCopy()
	deployByte, _ := json.Marshal(new)
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
