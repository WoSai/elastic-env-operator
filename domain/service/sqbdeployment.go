package service

import (
	"context"
	"errors"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	appv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type sqbDeploymentHandler struct {
	req ctrl.Request
	ctx context.Context
}

func NewSqbDeploymentHanlder(req ctrl.Request, ctx context.Context) *sqbApplicationHandler {
	return &sqbApplicationHandler{req: req, ctx: ctx}
}

func (h *sqbDeploymentHandler) GetInstance() (runtimeObj, error) {
	in := &entity.SQBDeployment{}
	if err := k8sclient.Get(h.ctx, h.req.NamespacedName, in); err != nil {
		return in, err
	}

	sqbapplication := &entity.SQBApplication{}
	if err := k8sclient.Get(h.ctx, client.ObjectKey{
		Namespace: in.Namespace,
		Name:      in.Spec.Selector.App,
	}, sqbapplication); err != nil {
		if apierrors.IsNotFound(err) {
			return in, errors.New("SQBApplication Not Found")
		}
		return in, err
	}
	in.SqbApplication = sqbapplication

	deployment := &appv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: in.Namespace,
			Name:      in.Name,
		},
	}
	if err := k8sclient.Get(h.ctx, h.req.NamespacedName, deployment); err != nil && apierrors.IsNotFound(err) {
		return in, err
	}
	in.Deployment = deployment

	if publicEntry, ok := in.Annotations[entity.PublicEntryAnnotationKey]; ok && publicEntry == "true" &&
		in.SqbApplication.IsIstioInject() {
		specialVirtualService := &istio.VirtualService{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: in.Namespace,
				Name:      in.Name,
			},
		}
		if err := k8sclient.Get(h.ctx, h.req.NamespacedName, specialVirtualService); err != nil && apierrors.IsNotFound(err) {
			return in, err
		}
		in.SpecialVirtualService = specialVirtualService
	}
	in.BuildRef()
	return in, nil
}

// 初始化逻辑
func (h *sqbDeploymentHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBDeployment)
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
func (h *sqbDeploymentHandler) Operate(obj runtimeObj) error {
	in := obj.(*entity.SQBDeployment)
	if err := CreateOrUpdate(h.ctx, in.Deployment); err != nil {
		return err
	}
	specialVirtualService := &istio.VirtualService{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: in.Namespace,
			Name:      in.Name,
		},
	}
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, specialVirtualService)

	if err != nil {
		if apierrors.IsNotFound(err) {
			if err := CreateOrUpdate(h.ctx, in.SpecialVirtualService); err != nil {
				return err
			}
		} else {
			return err
		}
	} else {
		if in.SpecialVirtualService == nil {
			if err := Delete(h.ctx, specialVirtualService); err != nil {
				return err
			}
		} else {
			if err := CreateOrUpdate(h.ctx, in.SpecialVirtualService); err != nil {
				return err
			}
		}
	}
	return nil
}

// 处理失败后逻辑
func (h *sqbDeploymentHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*entity.SQBDeployment)
	in.Status.ErrorInfo = err.Error()
	_ = k8sclient.Status().Update(h.ctx, in)
}

// 删除逻辑
func (h *sqbDeploymentHandler) IsDeleting(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBDeployment)
	if in.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(in, entity.SqbdeploymentFinalizer) {
		return false, nil
	}

	if deleteCheckSum, ok := in.Annotations[entity.ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(in.Name) {
		if err := Delete(h.ctx, in.Deployment); err != nil {
			return true, err
		}
	}
	return true, RemoveFinalizer(h.ctx, in, entity.SqbdeploymentFinalizer)
}
