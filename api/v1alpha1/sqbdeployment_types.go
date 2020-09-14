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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SQBDeploymentSpec defines the desired state of SQBDeployment
type SQBDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Selector   Selector `json:"selector"`
	DeploySpec `json:",inline"`
}

type Selector struct {
	App   string `json:"app"`
	Plane string `json:"plane"`
}

// SQBDeploymentStatus defines the observed state of SQBDeployment
type SQBDeploymentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Initialized bool   `json:"initialized,omitempty"`
	ErrorInfo   string `json:"errorInfo,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SQBDeployment is the Schema for the sqbdeployments API
type SQBDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SQBDeploymentSpec   `json:"spec,omitempty"`
	Status SQBDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SQBDeploymentList contains a list of SQBDeployment
type SQBDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQBDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SQBDeployment{}, &SQBDeploymentList{})
}