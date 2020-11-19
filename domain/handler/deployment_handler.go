package handler

import (
	"context"
	"encoding/json"
	"github.com/gogo/protobuf/proto"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	deployment.Spec.Template.Spec.TerminationGracePeriodSeconds = proto.Int64(300)
	deployment.Spec.Template.Spec.EnableServiceLinks = proto.Bool(false)

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
	// init lifecycle
	if deploy.Lifecycle != nil && deploy.Lifecycle.Init != nil {
		init := deploy.Lifecycle.Init
		image, ok := h.sqbdeployment.Annotations[entity.InitContainerAnnotationKey]
		if !ok {
			image = "busybox"
		}
		initContainer := corev1.Container{
			Name:         "init-1",
			Image:        image,
			Command:      init.Exec.Command,
			Env:          deploy.Env,
			VolumeMounts: deploy.VolumeMounts,
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
	return CreateOrUpdate(h.ctx, deployment)
}

func (h *deploymentHandler) Delete() error {
	deployment := &appv1.Deployment{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, deployment)
}

func (h *deploymentHandler) Handle() error {
	if deleted, _ := IsDeleted(h.sqbdeployment); deleted {
		return h.Delete()
	}
	if !h.sqbdeployment.DeletionTimestamp.IsZero() {
		return nil
	}
	return h.CreateOrUpdate()
}

func (h *deploymentHandler) GetInstance() (runtimeObj, error) {
	in := &appv1.Deployment{}
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
			mirrors[deployment.Name] = 1
			if p, ok := deployment.Labels[entity.PlaneKey]; ok {
				planes[p] = 1
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
			mirrors[deployment.Name] = 1
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

func (h *deploymentHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*appv1.Deployment)
	recorded := false
	for i:=0; i<len(in.Status.Conditions); i++ {
		if in.Status.Conditions[i].Type == appv1.DeploymentProgressing {
			in.Status.Conditions[i].Message = err.Error()
			in.Status.Conditions[i].LastUpdateTime = metav1.Now()
			recorded = true
			break
		}
	}
	if !recorded {
		in.Status.Conditions = append(in.Status.Conditions, appv1.DeploymentCondition{
			Type: appv1.DeploymentProgressing,
			Message: err.Error(),
			LastUpdateTime: metav1.Now(),
		})
	}
	_ = UpdateStatus(h.ctx, in)
}
