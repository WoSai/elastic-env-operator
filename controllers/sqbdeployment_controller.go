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
	types2 "github.com/gogo/protobuf/types"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	v1beta14 "istio.io/api/networking/v1beta1"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
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

// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbdeployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbdeployments/status,verbs=get;update;patch

func (r *SQBDeploymentReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	instance := &qav1alpha1.SQBDeployment{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return HandleReconcile(r, ctx, instance)
}

func (r *SQBDeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qav1alpha1.SQBDeployment{}, builder.WithPredicates(GenerationAnnotationPredicate)).
		Complete(r)
}

func (r *SQBDeploymentReconciler) IsInitialized(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBDeployment)
	var err error
	if cr.Status.Initialized == true {
		return true, nil
	}
	// 设置finalizer、labels
	controllerutil.AddFinalizer(cr, SqbdeploymentFinalizer)
	cr.Labels = addLabel(cr.Labels, AppKey, cr.Spec.Selector.App)
	cr.Labels = addLabel(cr.Labels, PlaneKey, cr.Spec.Selector.Plane)

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
	configMapData := getDefaultConfigMapData(r.Client, ctx)
	application := &qav1alpha1.SQBApplication{}
	err := r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: cr.Spec.Selector.App}, application)
	if err != nil {
		return err
	}
	plane := &qav1alpha1.SQBPlane{}
	err = r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: cr.Spec.Selector.Plane}, plane)
	if err != nil {
		return err
	}
	deploymentName := getSubsetName(cr.Spec.Selector.App, cr.Spec.Selector.Plane)

	deployment := &v12.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name: deploymentName,
		Namespace: cr.Namespace},
	}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		var err error
		applicationDeploy, _ := json.Marshal(application.Spec.DeploySpec)
		if globalDefaultDeploy, ok := configMapData["globalDefaultDeploy"]; ok {
			applicationDeploy, _ = jsonpatch.MergePatch([]byte(globalDefaultDeploy), applicationDeploy)
		}
		deploymentDeploy, _ := json.Marshal(cr.Spec.DeploySpec)
		mergeDeploy, _ := jsonpatch.MergePatch(applicationDeploy, deploymentDeploy)
		deploy := &qav1alpha1.DeploySpec{}
		err = json.Unmarshal(mergeDeploy, deploy)
		if err != nil {
			return err
		}
		// 组装deployment
		container := v1.Container{
			Name:           deploymentName,
			Image:          deploy.Image,
			Env:            deploy.Env,
			VolumeMounts:   deploy.VolumeMounts,
			LivenessProbe:  deploy.HealthCheck,
			ReadinessProbe: deploy.HealthCheck,
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

		if len(deployment.Labels) == 0 {
			deployment.Labels = map[string]string{}
		}
		deployment.Labels[AppKey] = cr.Spec.Selector.App
		deployment.Labels[PlaneKey] = cr.Spec.Selector.Plane

		deployment.Spec = v12.DeploymentSpec{
			Replicas: deploy.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					AppKey: cr.Spec.Selector.App,
				},
			},
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						AppKey:   cr.Spec.Selector.App,
						PlaneKey: cr.Spec.Selector.Plane,
					},
				},
				Spec: v1.PodSpec{
					Volumes: deploy.Volumes,
					Containers: []v1.Container{
						container,
					},
				},
			},
		}

		imagePullSecrets, ok := configMapData["imagePullSecrets"]
		if ok {
			deployment.Spec.Template.Spec.ImagePullSecrets = []v1.LocalObjectReference{{Name: imagePullSecrets}}
		}

		if anno, ok := cr.Annotations[PodAnnotationKey]; ok {
			err = json.Unmarshal([]byte(anno), &deployment.Spec.Template.Annotations)
			if err != nil {
				return err
			}
		}
		if len(deployment.Annotations) == 0 {
			deployment.Annotations = map[string]string{}
		}

		if anno, ok := cr.Annotations[DeploymentAnnotationKey]; ok {
			err = json.Unmarshal([]byte(anno), &deployment.Annotations)
			if err != nil {
				return err
			}
		}
		// sqbapplication controller要用到publicEntry
		if publicEntry, ok := cr.Annotations[PublicEntryAnnotationKey]; ok {
			deployment.Annotations[PublicEntryAnnotationKey] = publicEntry
		}

		// init lifecycle
		if deploy.Lifecycle != nil && deploy.Lifecycle.Init != nil {
			init := deploy.Lifecycle.Init
			initContainer := v1.Container{
				Name:    "busybox",
				Image:   "busybox",
				Command: init.Exec.Command,
				Env:     deploy.Env,
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
								Operator: "In",
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

		// 设置deployment的owner ref
		err = controllerutil.SetOwnerReference(application, deployment, r.Scheme)
		if err != nil {
			return err
		}
		err = controllerutil.SetOwnerReference(plane, deployment, r.Scheme)
		if err != nil {
			return err
		}
		return nil
	})

	if err != nil {
		return err
	}
	publicEntry, ok := cr.Annotations[PublicEntryAnnotationKey]
	if ok && isIstioEnable(r.Client, ctx, configMapData, application) {
		// 如果打开特殊入口，创建或更新单独的virtualservice
		virtualservice := r.generateSpecialVirtualService(deployment, configMapData, publicEntry)
		// 如果已经存在同名的virtualservice，就不再做变化，因为有可能手动修改过
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, virtualservice, func() error { return nil })
		if err != nil {
			return err
		}
	} else {
		// 如果没有打开特殊入口或没有启用istio，删除单独的virtualservice
		virtualservice := &v1beta1.VirtualService{}
		err = r.Get(ctx, types.NamespacedName{Namespace: cr.Namespace, Name: cr.Name}, virtualservice)
		if err == nil {
			err = r.Delete(ctx, virtualservice)
			if err != nil {
				return err
			}
		}
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

	configMapData := getDefaultConfigMapData(r.Client, ctx)

	// 如果configmap没有配置密码，直接删除资源
	password, ok := configMapData["deletePassword"]
	if !ok {
		return true, r.RemoveFinalizer(ctx, cr)
	}
	if cr.Annotations[ExplicitDeleteAnnotationKey] == "true" && cr.Annotations[DeletePasswordAnnotationKey] == password {
		// 删除对应Deployment
		deployment := &v12.Deployment{ObjectMeta: metav1.ObjectMeta{
			Name: getSubsetName(cr.Spec.Selector.App, cr.Spec.Selector.Plane),
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

func (r *SQBDeploymentReconciler) generateSpecialVirtualService(cr *v12.Deployment,
	configMapData map[string]string, host string) *v1beta1.VirtualService {
	virtualservice := &v1beta1.VirtualService{
		ObjectMeta: metav1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name},
		Spec: v1beta14.VirtualService{
			Hosts:    []string{host},
			Gateways: getIstioGateways(configMapData),
			Http: []*v1beta14.HTTPRoute{
				{
					Route: []*v1beta14.HTTPRouteDestination{
						{Destination: &v1beta14.Destination{
							Host:   cr.Labels[AppKey],
							Subset: cr.Name,
						}},
					},
					Timeout: &types2.Duration{Seconds: getIstioTimeout(configMapData)},
					Headers: &v1beta14.Headers{
						Request: &v1beta14.Headers_HeaderOperations{Set: map[string]string{XEnvFlag: cr.Labels[PlaneKey]}},
					},
				},
			},
		},
	}
	// 设置controller owner ref,删除Deployment自动删除
	_ = controllerutil.SetControllerReference(cr, virtualservice, r.Scheme)
	return virtualservice
}

func deleteSqbdeploymentByLabel(c client.Client, ctx context.Context, namespace string, labelSets map[string]string) error {
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

func deleteDeploymentByLabel(c client.Client, ctx context.Context, namespace string, labelSets map[string]string) error {
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

func getSpecialVirtualServiceHost(configMapData map[string]string, cr *v12.Deployment) string {
	publicEntry := cr.Annotations[PublicEntryAnnotationKey]
	return publicEntry
}
