package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	appv1 "k8s.io/api/apps/v1"
)

type deploymentListHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	sqbplane       *qav1alpha1.SQBPlane
	ctx            context.Context
}

func NewDeploymentListHandler(sqbapplication *qav1alpha1.SQBApplication, sqbplane *qav1alpha1.SQBPlane, ctx context.Context) *deploymentListHandler {
	return &deploymentListHandler{sqbapplication: sqbapplication, sqbplane: sqbplane, ctx: ctx}
}

func (h *deploymentListHandler) Handle() error {
	if h.sqbapplication != nil && IsExplicitDelete(h.sqbapplication) {
		return DeleteAllOf(h.ctx, &appv1.Deployment{}, h.sqbapplication.Namespace, map[string]string{entity.AppKey: h.sqbapplication.Name})
	}
	if h.sqbplane != nil && IsExplicitDelete(h.sqbplane) {
		return DeleteAllOf(h.ctx, &appv1.Deployment{}, h.sqbplane.Namespace, map[string]string{entity.PlaneKey: h.sqbplane.Name})
	}
	return nil
}
