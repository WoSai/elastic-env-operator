package handler

import (
	"context"
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

func NewSqbDeploymentHanlder(req ctrl.Request, ctx context.Context) *sqbDeploymentHandler {
	return &sqbDeploymentHandler{req: req, ctx: ctx}
}

func (h *sqbDeploymentHandler) GetInstance() (runtimeObj, error) {
	in := &entity.SQBDeploymentEntity{}
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, &in.SQBDeployment)
	return in, err
}

// 初始化逻辑
func (h *sqbDeploymentHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBDeploymentEntity)
	if in.Annotations[entity.InitializeAnnotationKey] == "true" {
		return true, nil
	}
	sqbapplication := &entity.SQBApplicationEntity{}
	if err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: in.Namespace, Name: in.Spec.Selector.App},
		&sqbapplication.SQBApplication); err != nil {
		return false, err
	}
	in.Initialize(&sqbapplication.SQBApplication)
	return false, CreateOrUpdate(h.ctx, &in.SQBDeployment)
}

// 正常处理逻辑
func (h *sqbDeploymentHandler) Operate(obj runtimeObj) error {
	in := obj.(*entity.SQBDeploymentEntity)
	objmeta := metav1.ObjectMeta{Namespace: in.Namespace, Name: in.Name}
	deployment := &appv1.Deployment{ObjectMeta: objmeta}
	if err := k8sclient.Get(h.ctx, h.req.NamespacedName, deployment); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	in.Deployment = deployment
	in.UpdateDeployment()
	if err := CreateOrUpdate(h.ctx, in.Deployment); err != nil {
		return err
	}

	if entity.ConfigMapData.IstioEnable() {
		specialVirtualService := &istio.VirtualService{ObjectMeta: objmeta}
		err := k8sclient.Get(h.ctx, h.req.NamespacedName, specialVirtualService)

		if in.HasPublicEntry() {
			in.SpecialVirtualService = specialVirtualService
			in.UpdateSpecialVirtualService()
			if err := CreateOrUpdate(h.ctx, in.SpecialVirtualService); err != nil {
				return err
			}
		} else if err == nil {
			if err := Delete(h.ctx, specialVirtualService); err != nil {
				return err
			}
		}
	}
	if in.Status.ErrorInfo != "" {
		in.Status.ErrorInfo = ""
	}
	if err := k8sclient.Status().Update(h.ctx, &in.SQBDeployment); err != nil {
		return err
	}
	return CreateOrUpdate(h.ctx, &in.SQBDeployment)
}

// 处理失败后逻辑
func (h *sqbDeploymentHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*entity.SQBDeploymentEntity)
	in.Status.ErrorInfo = err.Error()
	_ = k8sclient.Status().Update(h.ctx, &in.SQBDeployment)
}

// 删除逻辑
func (h *sqbDeploymentHandler) IsDeleting(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBDeploymentEntity)
	if in.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(in, entity.SqbdeploymentFinalizer) {
		return false, nil
	}
	if deleteCheckSum, ok := in.Annotations[entity.ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(in.Name) {
		if err := Delete(h.ctx, in.Deployment); err != nil {
			return true, err
		}
	}
	controllerutil.RemoveFinalizer(in, entity.SqbdeploymentFinalizer)
	return true, CreateOrUpdate(h.ctx, &in.SQBDeployment)
}
