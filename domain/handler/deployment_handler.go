package handler

import (
	"context"
	"encoding/json"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type deploymentHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx context.Context
}

func newDeploymentHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) *deploymentHandler {
	return &deploymentHandler{sqbdeployment: sqbdeployment, ctx: ctx}
}

func (h *deploymentHandler) CreateOrUpdate() error {
	deployment := &appv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: deployment.Namespace, Name: deployment.Name}, deployment)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	deploy := h.sqbdeployment.Spec.DeploySpec
	container := corev1.Container{
		Name:           h.sqbdeployment.Name,
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
	deployment.Spec.Template.Spec.Volumes = deploy.Volumes
	deployment.Spec.Template.Spec.HostAliases = deploy.HostAlias
	deployment.Spec.Template.Spec.Containers = []corev1.Container{container}
	deployment.Spec.Template.Spec.ImagePullSecrets = entity.ConfigMapData.GetImagePullSecrets()

	if anno, ok := h.sqbdeployment.Annotations[entity.PodAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &deployment.Spec.Template.Annotations)
	} else {
		deployment.Spec.Template.Annotations = nil
	}

	if anno, ok := h.sqbdeployment.Annotations[entity.DeploymentAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &deployment.Annotations)
	} else {
		deployment.Annotations = nil
	}
	if len(deployment.Annotations) == 0 {
		deployment.Annotations = make(map[string]string)
	}
	// sqbapplication controller要用到publicEntry
	if publicEntry, ok := h.sqbdeployment.Annotations[entity.PublicEntryAnnotationKey]; ok {
		deployment.Annotations[entity.PublicEntryAnnotationKey] = publicEntry
	} else {
		delete(deployment.Annotations, entity.PublicEntryAnnotationKey)
	}
	// init lifecycle
	if deploy.Lifecycle != nil && deploy.Lifecycle.Init != nil {
		init := deploy.Lifecycle.Init
		initContainer := corev1.Container{
			Name:         "busybox",
			Image:        "busybox",
			Command:      init.Exec.Command,
			Env:          deploy.Env,
			VolumeMounts: deploy.VolumeMounts,
		}
		deployment.Spec.Template.Spec.InitContainers = []corev1.Container{initContainer}
	}
	// NodeAffinity
	if deploy.NodeAffinity != nil {
		var nodeAffinity []corev1.PreferredSchedulingTerm
		for _, item := range deploy.NodeAffinity {
			nodeAffinity = append(nodeAffinity, corev1.PreferredSchedulingTerm{
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
		affinity := &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: nodeAffinity,
			},
		}
		deployment.Spec.Template.Spec.Affinity = affinity
	}
	return CreateOrUpdate(h.ctx, deployment)
}

func (h *deploymentHandler) Delete() error {
	deployment := &appv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, deployment)
}
