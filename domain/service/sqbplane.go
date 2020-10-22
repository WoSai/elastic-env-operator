package service

import (
	"context"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	appv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type sqbPlaneHandler struct {
	req ctrl.Request
	ctx context.Context
}

func NewSqbPlaneHanlder(req ctrl.Request, ctx context.Context) *sqbPlaneHandler {
	return &sqbPlaneHandler{req: req, ctx: ctx}
}

func (h *sqbPlaneHandler) GetInstance() (runtimeObj, error) {
	in := &entity.SQBPlane{}
	if err := k8sclient.Get(h.ctx, h.req.NamespacedName, in); err != nil {
		return in, err
	}
	if !in.Status.Initialized {
		return in, nil
	}

	deployments := &appv1.DeploymentList{}
	if err := k8sclient.List(h.ctx, deployments,
		&client.ListOptions{
			Namespace:     in.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{entity.PlaneKey: in.Name}),
		},
	); err != nil && apierrors.IsNotFound(err) {
		return in, err
	}
	in.Deployments = deployments

	mirrors := make(map[string]int)
	for _, deployment := range deployments.Items {
		mirrors[deployment.Name] = 1
	}
	in.Status.Mirrors = mirrors
	return in, nil
}

// 初始化逻辑
func (h *sqbPlaneHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBPlane)
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
func (h *sqbPlaneHandler) Operate(obj runtimeObj) error {
	in := obj.(*entity.SQBPlane)
	mirrors := make(map[string]int)
	for _, deployment := range in.Deployments.Items {
		mirrors[deployment.Name] = 1
	}
	in.Status.Mirrors = mirrors
	in.Status.ErrorInfo = ""
	return k8sclient.Status().Update(h.ctx, in)
}

// 处理失败后逻辑
func (h *sqbPlaneHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*entity.SQBPlane)
	in.Status.ErrorInfo = err.Error()
	_ = k8sclient.Status().Update(h.ctx, in)
}

// 删除逻辑
func (h *sqbPlaneHandler) IsDeleting(obj runtimeObj) (bool, error) {
	in := obj.(*entity.SQBPlane)
	if in.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(in, entity.SqbplaneFinalizer) {
		return false, nil
	}

	if deleteCheckSum, ok := in.Annotations[entity.ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(in.Name) {
		if err := k8sclient.DeleteAllOf(h.ctx, &entity.SQBDeployment{}, &client.DeleteAllOfOptions{
			ListOptions: client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.PlaneKey: in.Name}),
			},
		}); err != nil {
			return true, err
		}
		if err := k8sclient.DeleteAllOf(h.ctx, &appv1.Deployment{}, &client.DeleteAllOfOptions{
			ListOptions: client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.PlaneKey: in.Name}),
			},
		}); err != nil {
			return true, err
		}
	}
	return true, RemoveFinalizer(h.ctx, in, entity.SqbplaneFinalizer)
}
