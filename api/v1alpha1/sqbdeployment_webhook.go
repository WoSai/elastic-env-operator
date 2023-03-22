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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var sqbdeploymentlog = logf.Log.WithName("sqbdeployment-resource")

func (r *SQBDeployment) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=update,path=/validate-qa-shouqianba-com-v1alpha1-sqbdeployment,mutating=false,failurePolicy=fail,groups=qa.shouqianba.com,resources=sqbdeployments,versions=v1alpha1,name=vsqbdeployment.kb.io

var _ webhook.Validator = &SQBDeployment{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *SQBDeployment) ValidateCreate() error {
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *SQBDeployment) ValidateUpdate(old runtime.Object) error {
	sqbdeploymentlog.Info("validate update", "name", r.Name)
	oldcr := old.(*SQBDeployment)
	if r.Spec.Selector.App != oldcr.Spec.Selector.App ||
		r.Spec.Selector.Plane != oldcr.Spec.Selector.Plane {
		sqbdeploymentlog.Info("sqbdeployment selector updated", "name", r.Name)
	}
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *SQBDeployment) ValidateDelete() error {
	return nil
}
