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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/wosai/elastic-env-operator/domain/entity"
	v1 "k8s.io/api/core/v1"
	v14 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// SQBPlaneReconciler reconciles a SQBPlane object
type ConfigMapReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbplanes/status,verbs=get;update;patch

func (r *ConfigMapReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	instance := &v1.ConfigMap{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	data := instance.Data
	istio := &v14.CustomResourceDefinition{}
	err = r.Get(ctx, types.NamespacedName{Namespace: "", Name: "virtualservices.networking.istio.io"}, istio)
	if err == nil {
		data["istioEnable"] = "true"
	} else {
		data["istioEnable"] = "false"
	}
	entity.ConfigMapData.FromMap(data)
	return ctrl.Result{}, r.Update(ctx, instance)
}

func (r *ConfigMapReconciler) SetupWithManager(mgr ctrl.Manager) error {
	namespace := os.Getenv("CONFIGMAP_NAMESPACE")
	if namespace == "" {
		namespace = "elastic-env-operator-system"
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.ConfigMap{}, builder.WithPredicates(predicate.NewPredicateFuncs(
			func(meta metav1.Object, object runtime.Object) bool {
				if meta.GetNamespace() == namespace && meta.GetName() == "operator-configmap" {
					return true
				}
				return false
			}))).
		Complete(r)
}
