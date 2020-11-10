package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
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
	in := &qav1alpha1.SQBDeployment{}
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, in)
	return in, err
}

// 初始化逻辑
func (h *sqbDeploymentHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*qav1alpha1.SQBDeployment)
	if in.Annotations[entity.InitializeAnnotationKey] == "true" {
		return true, nil
	}
	sqbapplication := &qav1alpha1.SQBApplication{}
	if err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: in.Namespace, Name: in.Spec.Selector.App},
		sqbapplication); err != nil {
		return false, err
	}

	newSQBDeployment := &qav1alpha1.SQBDeployment{}
	newSQBDeployment.Spec.DeploySpec = sqbapplication.Spec.DeploySpec
	newSQBDeployment.Labels = sqbapplication.Labels

	in.Merge(newSQBDeployment)
	controllerutil.AddFinalizer(in, entity.Finalizer)
	in.Labels = util.MergeStringMap(in.Labels, map[string]string{
		entity.AppKey:   in.Spec.Selector.App,
		entity.PlaneKey: in.Spec.Selector.Plane,
	})
	if len(in.Annotations) == 0 {
		in.Annotations = make(map[string]string)
	}
	in.Annotations[entity.InitializeAnnotationKey] = "true"
	return false, CreateOrUpdate(h.ctx, in)
}

// 正常处理逻辑
func (h *sqbDeploymentHandler) Operate(obj runtimeObj) error {
	in := obj.(*qav1alpha1.SQBDeployment)
	if err := NewDeploymentHandler(in, h.ctx).CreateOrUpdate(); err != nil {
		return err
	}

	if entity.ConfigMapData.IstioEnable() {
		handler := NewSpecialVirtualServiceHandler(in, h.ctx)
		if in.Annotations[entity.PublicEntryAnnotationKey] == "true" {
			if err := handler.CreateOrUpdate(); err != nil {
				return err
			}
		} else {
			if err := handler.Delete(); err != nil {
				return err
			}
		}
	}
	in.Status.ErrorInfo = ""
	return k8sclient.Status().Update(h.ctx, in)
}

// 处理失败后逻辑
func (h *sqbDeploymentHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*qav1alpha1.SQBDeployment)
	in.Status.ErrorInfo = err.Error()
	_ = k8sclient.Status().Update(h.ctx, in)
}

// 删除逻辑
func (h *sqbDeploymentHandler) IsDeleting(obj runtimeObj) (bool, error) {
	in := obj.(*qav1alpha1.SQBDeployment)
	if in.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(in, entity.Finalizer) {
		return false, nil
	}
	if deleteCheckSum, ok := in.Annotations[entity.ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(in.Name) {
		if err := NewDeploymentHandler(in, h.ctx).Delete(); err != nil {
			return true, err
		}
	}
	controllerutil.RemoveFinalizer(in, entity.Finalizer)
	return true, CreateOrUpdate(h.ctx, in)
}

func HasPublicEntry(sqbdeployment *qav1alpha1.SQBDeployment) bool {
	publicEntry, ok := sqbdeployment.Annotations[entity.PublicEntryAnnotationKey]
	if ok {
		return publicEntry == "true"
	}
	return false
}