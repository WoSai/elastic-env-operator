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
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	"github.com/go-logr/logr"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	v12 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// SQBDeploymentReconciler reconciles a SQBDeployment object
type SQBDeploymentReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

var _ ISQBReconciler = &SQBDeploymentReconciler{}

// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbdeployments/status,verbs=get;update;patch

func (r *SQBDeploymentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	return HandleReconcile(r, ctx, req)
}

func (r *SQBDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qav1alpha1.SQBDeployment{}, builder.WithPredicates(GenerationAnnotationPredicate)).
		Complete(r)
}

func (r *SQBDeploymentReconciler) GetInstance(ctx context.Context, req ctrl.Request) (runtime.Object, error) {
	instance := &qav1alpha1.SQBDeployment{}
	err := r.Get(ctx, req.NamespacedName, instance)
	return instance, err
}

func (r *SQBDeploymentReconciler) IsInitialized(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBDeployment)
	var err error
	if cr.Status.Initialized == true {
		return true, nil
	}
	// 设置finalizer、labels
	controllerutil.AddFinalizer(cr, SqbdeploymentFinalizer)
	application := &qav1alpha1.SQBApplication{}
	err = r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: cr.Spec.Selector.App}, application)
	if err != nil {
		return false, err
	}

	applicationDeploy, _ := json.Marshal(application.Spec.DeploySpec)
	deploymentDeploy, _ := json.Marshal(cr.Spec.DeploySpec)
	mergeDeploy, _ := jsonpatch.MergePatch(applicationDeploy, deploymentDeploy)
	deploy := qav1alpha1.DeploySpec{}
	err = json.Unmarshal(mergeDeploy, &deploy)
	if err != nil {
		return false, err
	}
	cr.Spec.DeploySpec = deploy
	cr.Labels = util.MergeStringMap(application.Labels, cr.Labels)
	cr.Labels = util.MergeStringMap(cr.Labels, map[string]string{
		AppKey:   cr.Spec.Selector.App,
		PlaneKey: cr.Spec.Selector.Plane,
	})

	err = r.Update(ctx, cr)
	if err != nil {
		return false, err
	}
	// 更新status
	cr.Status.Initialized = true
	return false, r.Status().Update(ctx, cr)
}

func (r *SQBDeploymentReconciler) Operate(ctx context.Context, obj runtime.Object) error {
	cr := obj.(*qav1alpha1.SQBDeployment)
	deploymentName := util.GetSubsetName(cr.Spec.Selector.App, cr.Spec.Selector.Plane)

	deployment := &v12.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:      deploymentName,
		Namespace: cr.Namespace},
	}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		// 组装deployment
		deploy := cr.Spec.DeploySpec
		container := v1.Container{
			Name:           deploymentName,
			Image:          deploy.Image,
			Env:            deploy.Env,
			VolumeMounts:   deploy.VolumeMounts,
			LivenessProbe:  deploy.HealthCheck,
			ReadinessProbe: deploy.HealthCheck,
			Command:        deploy.Command,
			Args:           deploy.Args,
		}
		if deploy.Resources != nil {
			container.Resources = *deploy.Resources
		}
		if deploy.Lifecycle != nil {
			var lifecycle v1.Lifecycle
			if poststart := deploy.Lifecycle.PostStart; poststart != nil {
				lifecycle.PostStart = poststart
			}
			if prestop := deploy.Lifecycle.PreStop; prestop != nil {
				lifecycle.PreStop = prestop
			}
			container.Lifecycle = &lifecycle
		}

		deployment.Labels = cr.Labels
		deployment.Spec.Replicas = deploy.Replicas
		deployment.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: map[string]string{
				AppKey: cr.Spec.Selector.App,
			},
		}
		deployment.Spec.Template.ObjectMeta.Labels = deployment.Labels
		deployment.Spec.Template.Spec.Volumes = deploy.Volumes
		deployment.Spec.Template.Spec.HostAliases = deploy.HostAlias
		deployment.Spec.Template.Spec.Containers = []v1.Container{container}
		deployment.Spec.Template.Spec.ImagePullSecrets = entity.ConfigMapData.GetImagePullSecrets()

		if anno, ok := cr.Annotations[PodAnnotationKey]; ok {
			err := json.Unmarshal([]byte(anno), &deployment.Spec.Template.Annotations)
			if err != nil {
				return err
			}
		} else {
			deployment.Spec.Template.Annotations = nil
		}

		if anno, ok := cr.Annotations[DeploymentAnnotationKey]; ok {
			err := json.Unmarshal([]byte(anno), &deployment.Annotations)
			if err != nil {
				return err
			}
		} else {
			deployment.Annotations = nil
		}
		if len(deployment.Annotations) == 0 {
			deployment.Annotations = make(map[string]string)
		}
		// sqbapplication controller要用到publicEntry
		if publicEntry, ok := cr.Annotations[PublicEntryAnnotationKey]; ok {
			deployment.Annotations[PublicEntryAnnotationKey] = publicEntry
		} else {
			delete(deployment.Annotations, PublicEntryAnnotationKey)
		}
		// init lifecycle
		if deploy.Lifecycle != nil && deploy.Lifecycle.Init != nil {
			init := deploy.Lifecycle.Init
			initContainer := v1.Container{
				Name:         "busybox",
				Image:        "busybox",
				Command:      init.Exec.Command,
				Env:          deploy.Env,
				VolumeMounts: deploy.VolumeMounts,
			}
			deployment.Spec.Template.Spec.InitContainers = []v1.Container{initContainer}
		}
		// NodeAffinity
		if deploy.NodeAffinity != nil {
			var nodeAffinity []v1.PreferredSchedulingTerm
			for _, item := range deploy.NodeAffinity {
				nodeAffinity = append(nodeAffinity, v1.PreferredSchedulingTerm{
					Weight: item.Weight,
					Preference: v1.NodeSelectorTerm{
						MatchExpressions: []v1.NodeSelectorRequirement{
							{
								Key:      item.Key,
								Operator: item.Operator,
								Values:   item.Values,
							},
						},
					},
				})
			}
			affinity := &v1.Affinity{
				NodeAffinity: &v1.NodeAffinity{
					PreferredDuringSchedulingIgnoredDuringExecution: nodeAffinity,
				},
			}
			deployment.Spec.Template.Spec.Affinity = affinity
		}
		return nil
	})
	if err != nil {
		return err
	}
	cr.Status.ErrorInfo = ""
	return r.Status().Update(ctx, cr)
}

func (r *SQBDeploymentReconciler) IsDeleting(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBDeployment)
	if cr.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(cr, SqbdeploymentFinalizer) {
		return false, nil
	}
	var err error

	if deleteCheckSum, ok := cr.Annotations[ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(cr.Name) {
		deployment := &v12.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name:      util.GetSubsetName(cr.Spec.Selector.App, cr.Spec.Selector.Plane),
			Namespace: cr.Namespace},
		}
		err = r.Delete(ctx, deployment)
		if err != nil && !apierrors.IsNotFound(err) {
			return true, err
		}
	}
	return true, r.RemoveFinalizer(ctx, cr)
}

func (r *SQBDeploymentReconciler) ReconcileFail(ctx context.Context, obj runtime.Object, err error) {
	cr := obj.(*qav1alpha1.SQBDeployment)
	cr.Status.ErrorInfo = err.Error()
	_ = r.Status().Update(ctx, cr)
}

func (r *SQBDeploymentReconciler) RemoveFinalizer(ctx context.Context, cr *qav1alpha1.SQBDeployment) error {
	controllerutil.RemoveFinalizer(cr, SqbdeploymentFinalizer)
	return r.Update(ctx, cr)
}

func DeleteSqbdeploymentByLabel(c client.Client, ctx context.Context, namespace string, labelSets map[string]string) error {
	sqbDeploymentList := &qav1alpha1.SQBDeploymentList{}
	err := c.List(ctx, sqbDeploymentList, &client.ListOptions{Namespace: namespace, LabelSelector: labels.SelectorFromSet(labelSets)})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	for _, sqbDeployment := range sqbDeploymentList.Items {
		err = c.Delete(ctx, &sqbDeployment)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}

func DeleteDeploymentByLabel(c client.Client, ctx context.Context, namespace string, labelSets map[string]string) error {
	deploymentList := &v12.DeploymentList{}
	err := c.List(ctx, deploymentList, &client.ListOptions{Namespace: namespace, LabelSelector: labels.SelectorFromSet(labelSets)})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	for _, deployment := range deploymentList.Items {
		err = c.Delete(ctx, &deployment)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
