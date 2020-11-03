package handler

import (
	"context"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type sqbApplicationHandler struct {
	req ctrl.Request
	ctx context.Context
}

func NewSqbApplicationHanlder(req ctrl.Request, ctx context.Context) *sqbApplicationHandler {
	return &sqbApplicationHandler{req: req, ctx: ctx}
}

func (h *sqbApplicationHandler) GetInstance() (runtimeObj, error) {
	in := &entity.SQBApplicationEntity{}
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, &in.SQBApplication)
	return in, err
}

func (h *sqbApplicationHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBApplicationEntity)
	if in.Annotations[entity.InitializeAnnotationKey] == "true" {
		return true, nil
	}
	in.Initialize()
	return false, CreateOrUpdate(h.ctx, &in.SQBApplication)
}

func (h *sqbApplicationHandler) Operate(obj runtimeObj) error {
	in := obj.(*entity.SQBApplicationEntity)
	objmeta := metav1.ObjectMeta{Namespace: in.Namespace, Name: in.Name}
	service := &corev1.Service{ObjectMeta: objmeta}
	if err := k8sclient.Get(h.ctx, h.req.NamespacedName, service); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	in.Service = service
	in.UpdateService()
	if err := CreateOrUpdate(h.ctx, in.Service); err != nil {
		return err
	}

	deployments := &appv1.DeploymentList{}
	if err := k8sclient.List(h.ctx, deployments,
		&client.ListOptions{
			Namespace:     in.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: in.Name}),
		},
	); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	// SQBDeployment可能会被删除，所以planes和mirrors以deployment为准
	planes := make(map[string]int)
	mirrors := make(map[string]int)
	for _, deployment := range deployments.Items {
		mirrors[deployment.Name] = 1
		if plane, ok := deployment.Labels[entity.PlaneKey]; ok {
			planes[plane] = 1
		}
	}
	in.Status.Planes = planes
	in.Status.Mirrors = mirrors

	ingress := &v1beta1.Ingress{ObjectMeta: objmeta}
	if in.IsIngressOpen() {
		if err := k8sclient.Get(h.ctx, h.req.NamespacedName, ingress); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		in.Ingress = ingress
		in.UpdateIngress()
		if err := CreateOrUpdate(h.ctx, in.Ingress); err != nil {
			return err
		}
	} else {
		err := Delete(h.ctx, ingress)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	destinationrule := &istio.DestinationRule{ObjectMeta: objmeta}
	virtualservice := &istio.VirtualService{ObjectMeta: objmeta}
	if in.IsIstioInject() {
		if err := k8sclient.Get(h.ctx, h.req.NamespacedName, destinationrule); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		in.Destinationrule = destinationrule
		in.UpdateDestinationRule()
		if err := CreateOrUpdate(h.ctx, in.Destinationrule); err != nil {
			return err
		}
		if err := k8sclient.Get(h.ctx, h.req.NamespacedName, virtualservice); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		in.Virtualservice = virtualservice
		in.UpdateVirtualService()
		if err := CreateOrUpdate(h.ctx, in.Virtualservice); err != nil {
			return err
		}
	} else if entity.ConfigMapData.IstioEnable() {
		err := Delete(h.ctx, destinationrule)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		err = Delete(h.ctx, virtualservice)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	if in.Status.ErrorInfo != "" {
		in.Status.ErrorInfo = ""
	}
	if err := k8sclient.Status().Update(h.ctx, &in.SQBApplication); err != nil {
		return err
	}
	if len(planes) == 0 && in.Spec.Image != "" {
		// 创建对应的base环境服务
		sqbplane := entity.NewSQBPlane(in.Namespace, "base", "base")
		sqbdeployment := entity.NewSQBDeployment(in, sqbplane)
		if err := CreateOrUpdate(h.ctx, &sqbplane.SQBPlane); err != nil {
			return err
		}
		if err := CreateOrUpdate(h.ctx, &sqbdeployment.SQBDeployment); err != nil {
			return err
		}
	}
	return nil
}

func (h *sqbApplicationHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*entity.SQBApplicationEntity)
	in.Status.ErrorInfo = err.Error()
	_ = k8sclient.Status().Update(h.ctx, &in.SQBApplication)
}

func (h *sqbApplicationHandler) IsDeleting(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBApplicationEntity)
	if in.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(in, entity.SqbapplicationFinalizer) {
		return false, nil
	}
	if deleteCheckSum, ok := in.Annotations[entity.ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(in.Name) {
		objmeta := metav1.ObjectMeta{Namespace: in.Namespace, Name: in.Name}
		if err := Delete(h.ctx, &corev1.Service{ObjectMeta: objmeta}); err != nil {
			return true, err
		}
		if in.IsIngressOpen() {
			if err := Delete(h.ctx, &v1beta1.Ingress{ObjectMeta: objmeta}); err != nil {
				return true, err
			}
		}
		if in.IsIstioInject() {
			if err := Delete(h.ctx, &istio.VirtualService{ObjectMeta: objmeta}); err != nil {
				return true, err
			}
			if err := Delete(h.ctx, &istio.DestinationRule{ObjectMeta: objmeta}); err != nil {
				return true, err
			}
		}

		if err := k8sclient.DeleteAllOf(h.ctx, &entity.NewSQBDeployment(nil, nil).SQBDeployment, &client.DeleteAllOfOptions{
			ListOptions: client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: in.Name}),
			},
		}); err != nil {
			return true, err
		}
		if err := k8sclient.DeleteAllOf(h.ctx, &appv1.Deployment{}, &client.DeleteAllOfOptions{
			ListOptions: client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: in.Name}),
			},
		}); err != nil {
			return true, err
		}
	}
	controllerutil.RemoveFinalizer(in, entity.SqbapplicationFinalizer)
	return true, CreateOrUpdate(h.ctx, &in.SQBApplication)
}
