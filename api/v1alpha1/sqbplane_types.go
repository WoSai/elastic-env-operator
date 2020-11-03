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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SQBPlaneSpec defines the desired state of SQBPlane
type SQBPlaneSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Description string `json:"description,omitempty"`
}

// SQBPlaneStatus defines the observed state of SQBPlane
type SQBPlaneStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Mirrors     map[string]int `json:"mirrors,omitempty"`
	Initialized bool           `json:"initialized,omitempty"`
	ErrorInfo   string         `json:"errorInfo,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SQBPlane is the Schema for the sqbplanes API
type SQBPlane struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SQBPlaneSpec   `json:"spec,omitempty"`
	Status SQBPlaneStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SQBPlaneList contains a list of SQBPlane
type SQBPlaneList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SQBPlane `json:"items"`
}

func (old *SQBPlane) Merge(new *SQBPlane) {
	old.Annotations = util.MergeStringMap(old.Annotations, new.Annotations)
	old.Labels = util.MergeStringMap(old.Labels, new.Labels)
	old.Spec = new.Spec
}

func init() {
	SchemeBuilder.Register(&SQBPlane{}, &SQBPlaneList{})
}
