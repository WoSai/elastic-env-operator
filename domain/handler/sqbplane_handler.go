package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	ctrl "sigs.k8s.io/controller-runtime"
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
	in := &qav1alpha1.SQBPlane{}
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, in)
	return in, err
}

// 初始化逻辑
func (h *sqbPlaneHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*qav1alpha1.SQBPlane)
	if in.Annotations[entity.InitializeAnnotationKey] == "true" {
		return true, nil
	}
	controllerutil.AddFinalizer(in, entity.Finalizer)
	if len(in.Annotations) == 0 {
		in.Annotations = make(map[string]string)
	}
	in.Annotations[entity.InitializeAnnotationKey] = "true"
	return false, CreateOrUpdate(h.ctx, in)
}

// 正常处理逻辑
func (h *sqbPlaneHandler) Operate(obj runtimeObj) error {
	in := obj.(*qav1alpha1.SQBPlane)

	handlers := []SQBHandler{
		NewSqbDeploymentListHandler(nil, in, h.ctx),
		NewDeploymentListHandler(nil, in, h.ctx),
	}

	for _, handler := range handlers {
		if err := handler.Handle(); err != nil {
			return err
		}
	}

	if !in.DeletionTimestamp.IsZero() {
		controllerutil.RemoveFinalizer(in, entity.Finalizer)
		return CreateOrUpdate(h.ctx, in)
	} else if in.Status.ErrorInfo != "" {
		in.Status.ErrorInfo = ""
		return UpdateStatus(h.ctx, in)
	}
	return nil
}

// 处理失败后逻辑
func (h *sqbPlaneHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*qav1alpha1.SQBPlane)
	in.Status.ErrorInfo = err.Error()
	_ = UpdateStatus(h.ctx, in)
}
