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

func NewSqbDeploymentListHandlerForSqbapplication(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *sqbDeploymentListHandler {
	return &sqbDeploymentListHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func NewSqbDeploymentListHandlerForSqbplane(sqbplane *qav1alpha1.SQBPlane, ctx context.Context) *sqbDeploymentListHandler {
	return &sqbDeploymentListHandler{sqbplane: sqbplane, ctx: ctx}
}

func (h *sqbDeploymentListHandler) DeleteForSqbapplication() error {
	deleted, err := IsDeleted(h.sqbapplication)
	if err != nil {
		return err
	}
	if deleted {
		return DeleteAllOf(h.ctx, &qav1alpha1.SQBDeployment{}, h.sqbapplication.Namespace, map[string]string{entity.AppKey: h.sqbapplication.Name})
	}
	return nil
}

func (h *sqbDeploymentListHandler) DeleteForSqbplane() error {
	if deleted, _ := IsDeleted(h.sqbplane); deleted {
		return DeleteAllOf(h.ctx, &qav1alpha1.SQBDeployment{}, h.sqbplane.Namespace, map[string]string{entity.PlaneKey: h.sqbplane.Name})
	}
	return nil
}

func (h *sqbDeploymentListHandler) Handle() error {
	if h.sqbapplication != nil {
		return h.DeleteForSqbapplication()
	}
	if h.sqbplane != nil {
		return h.DeleteForSqbplane()
	}
	return nil
}
