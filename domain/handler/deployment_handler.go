package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/imdario/mergo"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

type deploymentHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx           context.Context
	req           ctrl.Request
}

func NewDeploymentHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) *deploymentHandler {
	return &deploymentHandler{sqbdeployment: sqbdeployment, ctx: ctx}
}

func NewDeploymentHandlerWithReq(req ctrl.Request, ctx context.Context) *deploymentHandler {
	return &deploymentHandler{req: req, ctx: ctx}
}

func (h *deploymentHandler) CreateOrUpdate() error {
	deployment := &appv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: deployment.Namespace, Name: deployment.Name}, deployment)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	deploy := h.sqbdeployment.Spec.DeploySpec
	volumes, volumeMounts := h.getVolumeAndVolumeMounts(deploy.Volumes)
	container := corev1.Container{
		Name:           h.sqbdeployment.Name,
		Image:          deploy.Image,
		Env:            deploy.Env,
		VolumeMounts:   volumeMounts,
		LivenessProbe:  deploy.HealthCheck,
		ReadinessProbe: deploy.HealthCheck,
		Command:        deploy.Command,
		Args:           deploy.Args,
	}
	if deploy.Resources != nil {
		container.Resources = corev1.ResourceRequirements{
			Requests: deploy.Resources.Requests,
			Limits:   deploy.Resources.Limits,
		}
	}
	if deploy.Lifecycle != nil {
		var lifecycle corev1.Lifecycle
		if poststart := deploy.Lifecycle.PostStart; poststart != nil {
			lifecycle.PostStart = poststart
		}
		if prestop := deploy.Lifecycle.PreStop; prestop != nil {
			lifecycle.PreStop = prestop
		}
		container.Lifecycle = &lifecycle
	}

	deployment.Labels = h.sqbdeployment.Labels
	deployment.Spec.Replicas = deploy.Replicas
	deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			entity.AppKey: h.sqbdeployment.Spec.Selector.App,
		},
	}
	deployment.Spec.Template.ObjectMeta.Labels = deployment.Labels
	deployment.Spec.Template.Spec.Volumes = volumes
	deployment.Spec.Template.Spec.HostAliases = deploy.HostAlias
	deployment.Spec.Template.Spec.Containers = []corev1.Container{container}
	deployment.Spec.Template.Spec.ImagePullSecrets = entity.ConfigMapData.GetImagePullSecrets()

	if anno, ok := h.sqbdeployment.Annotations[entity.PodAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &deployment.Spec.Template.Annotations)
	}

	deployment.Spec.Template.Annotations = util.MergeStringMap(deployment.Spec.Template.Annotations,
		map[string]string{entity.IstioSidecarInjectKey: h.sqbdeployment.Annotations[entity.IstioInjectAnnotationKey]})

	if anno, ok := h.sqbdeployment.Annotations[entity.DeploymentAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &deployment.Annotations)
	}
	// init lifecycle
	if deploy.Lifecycle != nil && deploy.Lifecycle.Init != nil {
		init := deploy.Lifecycle.Init
		image, ok := h.sqbdeployment.Annotations[entity.InitContainerAnnotationKey]
		if !ok {
			image = "busybox:1.32"
		}
		initContainer := corev1.Container{
			Name:            "init-1",
			Image:           image,
			Command:         init.Exec.Command,
			Env:             deploy.Env,
			VolumeMounts:    volumeMounts,
			ImagePullPolicy: corev1.PullIfNotPresent,
		}
		deployment.Spec.Template.Spec.InitContainers = []corev1.Container{initContainer}
	}
	// NodeAffinity
	if deploy.NodeAffinity != nil {
		affinity := &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{},
		}
		if len(deploy.NodeAffinity.Require) != 0 {
			nodeSelectorTerms := make([]corev1.NodeSelectorTerm, 0)
			for _, item := range deploy.NodeAffinity.Require {
				nodeSelectorTerms = append(nodeSelectorTerms, corev1.NodeSelectorTerm{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      item.Key,
							Operator: item.Operator,
							Values:   item.Values,
						},
					},
				})
			}
			affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &corev1.NodeSelector{
				NodeSelectorTerms: nodeSelectorTerms,
			}
		}
		if len(deploy.NodeAffinity.Prefer) != 0 {
			preferredTerms := make([]corev1.PreferredSchedulingTerm, 0)
			for _, item := range deploy.NodeAffinity.Prefer {
				preferredTerms = append(preferredTerms, corev1.PreferredSchedulingTerm{
					Weight: item.Weight,
					Preference: corev1.NodeSelectorTerm{
						MatchExpressions: []corev1.NodeSelectorRequirement{
							{
								Key:      item.Key,
								Operator: item.Operator,
								Values:   item.Values,
							},
						},
					},
				})
			}
			affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferredTerms
		}
		deployment.Spec.Template.Spec.Affinity = affinity
	}
	controllerutil.AddFinalizer(deployment, entity.FINALIZER)
	if specString := entity.ConfigMapData.DeploymentSpec(); specString != "" {
		if err = h.merge(deployment, specString); err != nil {
			return err
		}
	}
	return CreateOrUpdate(h.ctx, deployment)
}

func (h *deploymentHandler) merge(deployment *appv1.Deployment, specString string) error {
	spec := &appv1.DeploymentSpec{}
	if err := json.Unmarshal([]byte(specString), spec); err != nil {
		return err
	}
	if err := mergo.Merge(&deployment.Spec, spec); err != nil {
		return err
	}
	return nil
}

func (h *deploymentHandler) getVolumeAndVolumeMounts(volumemap []*qav1alpha1.VolumeSpec) (volumes []corev1.Volume, volumeMounts []corev1.VolumeMount) {
	for i, volumeSpec := range volumemap {
		volumeName := fmt.Sprintf("volume-%d", i)
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      volumeName,
			MountPath: volumeSpec.MountPath,
		})
		if volumeSpec.EmptyDir {
			volumes = append(volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			})
			continue
		}
		if volumeSpec.HostPath != "" {
			volumes = append(volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: volumeSpec.HostPath,
					},
				},
			})
			continue
		}
		if volumeSpec.ConfigMap != "" {
			volumes = append(volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: volumeSpec.ConfigMap,
						},
					},
				},
			})
			continue
		}
		if volumeSpec.Secret != "" {
			volumes = append(volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: volumeSpec.Secret,
					},
				},
			})
			continue
		}
		if volumeSpec.PersistentVolumeClaimName != "" {
			volumes = append(volumes, corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: volumeSpec.PersistentVolumeClaimName,
					},
				},
			})
			continue
		}
	}
	return
}

func (h *deploymentHandler) Delete() error {
	deployment := &appv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, deployment)
}

func (h *deploymentHandler) Handle() error {
	if deleted, _ := IsDeleted(h.sqbdeployment); deleted {
		return h.Delete()
	}
	return h.CreateOrUpdate()
}

func (h *deploymentHandler) GetInstance() (runtimeObj, error) {
	in := &appv1.Deployment{}
	time.Sleep(200 * time.Millisecond) // 很奇怪，predicate过滤的是有deletionTimestamp的，但是取出来的deployment确没有，等200ms之后取出来才有
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, in)
	return in, err
}

func (h *deploymentHandler) IsInitialized(_ runtimeObj) (bool, error) {
	return true, nil
}

func (h *deploymentHandler) Operate(obj runtimeObj) error {
	in := obj.(*appv1.Deployment)
	app := in.Labels[entity.AppKey]
	plane := in.Labels[entity.PlaneKey]
	// 更新sqbapplication的status
	deployments := &appv1.DeploymentList{}
	if app != "" {
		if err := k8sclient.List(h.ctx, deployments,
			&client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: app}),
			},
		); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		planes := make(map[string]int)
		mirrors := make(map[string]int)
		for _, deployment := range deployments.Items {
			if deployment.DeletionTimestamp.IsZero() {
				mirrors[deployment.Name] = 1
				if p, ok := deployment.Labels[entity.PlaneKey]; ok {
					planes[p] = 1
				}
			}
		}
		sqbapplication := &qav1alpha1.SQBApplication{}
		err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: in.Namespace, Name: app}, sqbapplication)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		} else {
			if sqbapplication.DeletionTimestamp.IsZero() {
				sqbapplication.Status.Planes = planes
				sqbapplication.Status.Mirrors = mirrors
				sqbapplication.Status.ErrorInfo = ""
				if err = UpdateStatus(h.ctx, sqbapplication); err != nil {
					return err
				}
			}
		}
	}
	// 更新sqbplane的status
	if plane != "" {
		if err := k8sclient.List(h.ctx, deployments,
			&client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.PlaneKey: plane}),
			},
		); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		mirrors := make(map[string]int)
		for _, deployment := range deployments.Items {
			if deployment.DeletionTimestamp.IsZero() {
				mirrors[deployment.Name] = 1
			}
		}
		sqbplane := &qav1alpha1.SQBPlane{}
		err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: in.Namespace, Name: plane}, sqbplane)
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		} else {
			if sqbplane.DeletionTimestamp.IsZero() {
				sqbplane.Status.Mirrors = mirrors
				sqbplane.Status.ErrorInfo = ""
				if err = UpdateStatus(h.ctx, sqbplane); err != nil {
					return err
				}
			}
		}
	}
	if !in.DeletionTimestamp.IsZero() {
		controllerutil.RemoveFinalizer(in, entity.FINALIZER)
		return CreateOrUpdate(h.ctx, in)
	}
	return nil
}

func (h *deploymentHandler) ReconcileFail(_ runtimeObj, _ error) {
	return
}
