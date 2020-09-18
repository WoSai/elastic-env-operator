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
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	v12 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// SQBPlaneReconciler reconciles a SQBPlane object
type SQBPlaneReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var _ ISQBReconciler = &SQBPlaneReconciler{}

// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbplanes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbplanes/status,verbs=get;update;patch

func (r *SQBPlaneReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	return HandleReconcile(r, ctx, req)
}

func (r *SQBPlaneReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qav1alpha1.SQBPlane{}, builder.WithPredicates(GenerationAnnotationPredicate)).
		Watches(&source.Kind{Type: &v12.Deployment{}},
			&handler.EnqueueRequestForOwner{OwnerType: &qav1alpha1.SQBPlane{}, IsController: false},
			builder.WithPredicates(CreateDeletePredicate)).
		Complete(r)
}

func (r *SQBPlaneReconciler) GetInstance(ctx context.Context, req ctrl.Request) (runtime.Object, error) {
	instance := &qav1alpha1.SQBPlane{}
	err := r.Get(ctx, req.NamespacedName, instance)
	return instance, client.IgnoreNotFound(err)
}

func (r *SQBPlaneReconciler) IsInitialized(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBPlane)
	if cr.Status.Initialized == true {
		return true, nil
	}
	controllerutil.AddFinalizer(cr, SqbplaneFinalizer)
	err := r.Update(ctx, cr)
	if err != nil {
		return false, err
	}
	cr.Status.Initialized = true
	return false, r.Status().Update(ctx, cr)
}

func (r *SQBPlaneReconciler) IsDeleting(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBPlane)
	if cr.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(cr, SqbplaneFinalizer) {
		return false, nil
	}

	var err error

	if deleteCheckSum, ok := cr.Annotations[ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == GetDeleteCheckSum(cr) {
		err = DeleteSqbdeploymentByLabel(r.Client, ctx, cr.Namespace, map[string]string{PlaneKey: cr.Name})
		if err != nil {
			return true, err
		}
		err = DeleteDeploymentByLabel(r.Client, ctx, cr.Namespace, map[string]string{PlaneKey: cr.Name})
		if err != nil {
			return true, err
		}
	}
	return true, r.RemoveFinalizer(ctx, cr)
}

func (r *SQBPlaneReconciler) Operate(ctx context.Context, obj runtime.Object) error {
	cr := obj.(*qav1alpha1.SQBPlane)
	var err error
	deploymentList := &v12.DeploymentList{}
	err = r.List(ctx, deploymentList, &client.ListOptions{Namespace: cr.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{PlaneKey: cr.Name})})
	if err != nil {
		return err
	}
	mirrors := map[string]int{}
	for _, deployment := range deploymentList.Items {
		mirrors[deployment.Name] = 1
	}
	cr.Status.Mirrors = mirrors
	cr.Status.ErrorInfo = ""
	return r.Status().Update(ctx, cr)
}

func (r *SQBPlaneReconciler) ReconcileFail(ctx context.Context, obj runtime.Object, err error) {
	cr := obj.(*qav1alpha1.SQBPlane)
	cr.Status.ErrorInfo = err.Error()
	_ = r.Status().Update(ctx, obj)
}

func (r *SQBPlaneReconciler) RemoveFinalizer(ctx context.Context, cr *qav1alpha1.SQBPlane) error {
	controllerutil.RemoveFinalizer(cr, SqbplaneFinalizer)
	return r.Update(ctx, cr)
}
