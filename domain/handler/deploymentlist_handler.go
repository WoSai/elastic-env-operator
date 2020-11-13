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

func NewDeploymentListHandlerForSqbapplication(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *deploymentListHandler {
	return &deploymentListHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func NewDeploymentListHandlerForSqbplane(sqbplane *qav1alpha1.SQBPlane, ctx context.Context) *deploymentListHandler {
	return &deploymentListHandler{sqbplane: sqbplane, ctx: ctx}
}

func (h *deploymentListHandler) DeleteForSqbapplication() error {
	if deleted, _ := IsDeleted(h.sqbapplication); deleted {
		return DeleteAllOf(h.ctx, &appv1.Deployment{}, h.sqbapplication.Namespace, map[string]string{entity.AppKey: h.sqbapplication.Name})
	}
	return nil
}

func (h *deploymentListHandler) DeleteForSqbplane() error {
	if deleted, _ := IsDeleted(h.sqbplane); deleted {
		return DeleteAllOf(h.ctx, &appv1.Deployment{}, h.sqbplane.Namespace, map[string]string{entity.PlaneKey: h.sqbplane.Name})
	}
	return nil
}

func (h *deploymentListHandler) Handle() error {
	if h.sqbapplication != nil {
		return h.DeleteForSqbapplication()
	}
	if h.sqbplane != nil {
		return h.DeleteForSqbplane()
	}
	return nil
}
