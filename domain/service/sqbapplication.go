package service

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
	in := &entity.SQBApplication{}
	if err := k8sclient.Get(h.ctx, h.req.NamespacedName, in); err != nil {
		return in, err
	}
	if !in.Status.Initialized {
		// 如果SQBApplication没有初始化，直接返回，不需要查下面的，之后会reenqueue
		return in, nil
	}

	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: in.Namespace,
			Name:      in.Name,
		},
	}
	if err := k8sclient.Get(h.ctx, h.req.NamespacedName, service); err != nil && apierrors.IsNotFound(err) {
		return in, err
	}
	in.Service = service

	deployments := &appv1.DeploymentList{}
	if err := k8sclient.List(h.ctx, deployments,
		&client.ListOptions{
			Namespace:     in.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: in.Name}),
		},
	); err != nil && apierrors.IsNotFound(err) {
		return in, err
	}
	in.Deployments = deployments
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

	if in.IsIngressOpen() {
		ingress := &v1beta1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: in.Namespace,
				Name:      in.Name,
			},
		}
		if err := k8sclient.Get(h.ctx, h.req.NamespacedName, ingress); err != nil && apierrors.IsNotFound(err) {
			return in, err
		}
		in.Ingress = ingress
	}

	if in.IsIstioInject() {
		destinationrule := &istio.DestinationRule{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: in.Namespace,
				Name:      in.Name,
			},
		}
		if err := k8sclient.Get(h.ctx, h.req.NamespacedName, destinationrule); err != nil && apierrors.IsNotFound(err) {
			return in, err
		}
		in.Destinationrule = destinationrule
		virtualservice := &istio.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: in.Namespace,
				Name:      in.Name,
			},
		}
		if err := k8sclient.Get(h.ctx, h.req.NamespacedName, virtualservice); err != nil && apierrors.IsNotFound(err) {
			return in, err
		}
		in.Virtualservice = virtualservice
	}
	in.BuildRef()
	return in, nil
}

// 初始化逻辑
func (h *sqbApplicationHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBApplication)
	if in.Status.Initialized {
		return true, nil
	}
	in.BuildSelf()
	if err := CreateOrUpdate(h.ctx, in); err != nil {
		return false, err
	}
	return false, k8sclient.Status().Update(h.ctx, in)
}

// 正常处理逻辑
func (h *sqbApplicationHandler) Operate(obj runtimeObj) error {
	in := obj.(*entity.SQBApplication)
	if err := CreateOrUpdate(h.ctx, in.Sqbplane); err != nil {
		return err
	}
	if err := CreateOrUpdate(h.ctx, in.Sqbdeployment); err != nil {
		return err
	}
	if err := CreateOrUpdate(h.ctx, in.Service); err != nil {
		return err
	}
	if err := CreateOrUpdate(h.ctx, in.Ingress); err != nil {
		return err
	}
	if err := CreateOrUpdate(h.ctx, in.Virtualservice); err != nil {
		return err
	}
	if err := CreateOrUpdate(h.ctx, in.Destinationrule); err != nil {
		return err
	}
	return k8sclient.Status().Update(h.ctx, in)
}

// 处理失败后逻辑
func (h *sqbApplicationHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*entity.SQBApplication)
	in.Status.ErrorInfo = err.Error()
	_ = k8sclient.Status().Update(h.ctx, in)
}

// 删除逻辑
func (h *sqbApplicationHandler) IsDeleting(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBApplication)
	if in.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(in, entity.SqbapplicationFinalizer) {
		return false, nil
	}

	if deleteCheckSum, ok := in.Annotations[entity.ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(in.Name) {
		// 删除ingress,service
		if err := Delete(h.ctx, in.Service); err != nil {
			return true, err
		}
		if err := Delete(h.ctx, in.Ingress); err != nil {
			return true, err
		}
		if err := Delete(h.ctx, in.Virtualservice); err != nil {
			return true, err
		}
		if err := Delete(h.ctx, in.Destinationrule); err != nil {
			return true, err
		}
		if err := k8sclient.DeleteAllOf(h.ctx, &entity.SQBDeployment{}, &client.DeleteAllOfOptions{
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
	return true, RemoveFinalizer(h.ctx, in, entity.SqbapplicationFinalizer)
}
