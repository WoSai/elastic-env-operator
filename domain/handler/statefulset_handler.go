package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gogo/protobuf/proto"
	"github.com/imdario/mergo"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

type statefulsetHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx           context.Context
	req           ctrl.Request
}

func NewStatefulsetHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) *statefulsetHandler {
	return &statefulsetHandler{sqbdeployment: sqbdeployment, ctx: ctx}
}

func NewStatefulsetHandlerWithReq(req ctrl.Request, ctx context.Context) *statefulsetHandler {
	return &statefulsetHandler{req: req, ctx: ctx}
}

func (h *statefulsetHandler) CreateOrUpdate() error {
	statefulset := &appv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: statefulset.Namespace, Name: statefulset.Name}, statefulset)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	deploy := h.sqbdeployment.Spec.DeploySpec
	volumes, pvcs, volumeMounts := h.getVolumeAndPvctempAndVolumeMounts(deploy.Volumes)
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

	statefulset.Labels = h.sqbdeployment.Labels
	statefulset.Spec.Replicas = deploy.Replicas
	statefulset.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			entity.AppKey: h.sqbdeployment.Spec.Selector.App,
		},
	}
	statefulset.Spec.VolumeClaimTemplates = pvcs
	statefulset.Spec.Template.ObjectMeta.Labels = statefulset.Labels
	statefulset.Spec.Template.Spec.Volumes = volumes
	statefulset.Spec.Template.Spec.HostAliases = deploy.HostAlias
	statefulset.Spec.Template.Spec.Containers = []corev1.Container{container}
	statefulset.Spec.Template.Spec.ImagePullSecrets = entity.ConfigMapData.GetImagePullSecrets()

	if anno, ok := h.sqbdeployment.Annotations[entity.PodAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &statefulset.Spec.Template.Annotations)
	}

	statefulset.Spec.Template.Annotations = util.MergeStringMap(statefulset.Spec.Template.Annotations,
		map[string]string{entity.IstioSidecarInjectKey: h.sqbdeployment.Annotations[entity.IstioInjectAnnotationKey]})

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
			VolumeMounts: volumeMounts,
		}
		statefulset.Spec.Template.Spec.InitContainers = []corev1.Container{initContainer}
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
		statefulset.Spec.Template.Spec.Affinity = affinity
	}
	controllerutil.AddFinalizer(statefulset, entity.FINALIZER)
	if specString := entity.ConfigMapData.StatefulsetSpec(); specString != "" {
		if err = h.merge(statefulset, specString); err != nil {
			return err
		}
	}
	return CreateOrUpdate(h.ctx, statefulset)
}

func (h *statefulsetHandler) merge(statefulset *appv1.StatefulSet, specString string) error {
	spec := &appv1.StatefulSetSpec{}
	if err := json.Unmarshal([]byte(specString), spec); err != nil {
		return err
	}
	if err := mergo.Merge(&statefulset.Spec, spec); err != nil {
		return err
	}
	return nil
}

func (h *statefulsetHandler) getVolumeAndPvctempAndVolumeMounts(volumemap []*qav1alpha1.VolumeSpec) (volumes []corev1.Volume, pvcs []corev1.PersistentVolumeClaim, volumeMounts []corev1.VolumeMount) {
	for i, volumeSpec := range volumemap {
		volumeName := fmt.Sprintf("volume-%d", i)

		if volumeSpec.PersistentVolumeClaim {
			pvcName := h.sqbdeployment.Name + "-" + volumeName
			pvcs = append(pvcs, corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name: pvcName,
				},
				Spec: corev1.PersistentVolumeClaimSpec{
					AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteMany},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceStorage: resource.MustParse("2Gi"),
						},
					},
					StorageClassName: proto.String("ack" + "-" + h.sqbdeployment.Labels[entity.GroupKey]),
				},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      pvcName,
				MountPath: volumeSpec.MountPath,
			})
			continue
		}

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
	}
	return
}

func (h *statefulsetHandler) Delete() error {
	statefulset := &appv1.StatefulSet{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, statefulset)
}

func (h *statefulsetHandler) Handle() error {
	if h.sqbdeployment.Annotations[entity.StatefulsetAnnotationKey] != "true" {
		return h.Delete()
	}
	if deleted, _ := IsDeleted(h.sqbdeployment); deleted {
		return h.Delete()
	}
	return h.CreateOrUpdate()
}

func (h *statefulsetHandler) GetInstance() (runtimeObj, error) {
	in := &appv1.StatefulSet{}
	time.Sleep(200 * time.Millisecond)
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, in)
	return in, err
}

func (h *statefulsetHandler) IsInitialized(_ runtimeObj) (bool, error) {
	return true, nil
}

func (h *statefulsetHandler) Operate(obj runtimeObj) error {
	in := obj.(*appv1.StatefulSet)
	app := in.Labels[entity.AppKey]
	plane := in.Labels[entity.PlaneKey]
	// 更新sqbapplication的status
	statefulsets := &appv1.StatefulSetList{}
	if app != "" {
		if err := k8sclient.List(h.ctx, statefulsets,
			&client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: app}),
			},
		); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		planes := make(map[string]int)
		mirrors := make(map[string]int)
		for _, statefulset := range statefulsets.Items {
			if statefulset.DeletionTimestamp.IsZero() {
				mirrors[statefulset.Name] = 1
				if p, ok := statefulset.Labels[entity.PlaneKey]; ok {
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
		if err := k8sclient.List(h.ctx, statefulsets,
			&client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.PlaneKey: plane}),
			},
		); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		mirrors := make(map[string]int)
		for _, statefulset := range statefulsets.Items {
			if statefulset.DeletionTimestamp.IsZero() {
				mirrors[statefulset.Name] = 1
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

func (h *statefulsetHandler) ReconcileFail(_ runtimeObj, _ error) {
	return
}
