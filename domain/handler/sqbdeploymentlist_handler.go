package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
)

type sqbDeploymentListHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	sqbplane       *qav1alpha1.SQBPlane
	ctx            context.Context
}

func NewSqbDeploymentListHandler(sqbapplication *qav1alpha1.SQBApplication, sqbplane *qav1alpha1.SQBPlane, ctx context.Context) *sqbDeploymentListHandler {
	return &sqbDeploymentListHandler{sqbapplication: sqbapplication, sqbplane: sqbplane, ctx: ctx}
}

func (h *sqbDeploymentListHandler) Handle() error {
	if h.sqbapplication != nil && IsExplicitDelete(h.sqbapplication) {
		return DeleteAllOf(h.ctx, &qav1alpha1.SQBDeployment{}, h.sqbapplication.Namespace, map[string]string{entity.AppKey: h.sqbapplication.Name})
	}
	if h.sqbplane != nil && IsExplicitDelete(h.sqbplane) {
		return DeleteAllOf(h.ctx, &qav1alpha1.SQBDeployment{}, h.sqbplane.Namespace, map[string]string{entity.PlaneKey: h.sqbplane.Name})
	}
	return nil
}
