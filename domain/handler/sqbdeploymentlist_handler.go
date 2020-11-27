package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	if deleted, _ := IsDeleted(h.sqbapplication); deleted {
		return h.deleteByLabel(map[string]string{entity.AppKey: h.sqbapplication.Name})
	}
	return nil
}

func (h *sqbDeploymentListHandler) DeleteForSqbplane() error {
	if deleted, _ := IsDeleted(h.sqbplane); deleted {
		return h.deleteByLabel(map[string]string{entity.PlaneKey: h.sqbplane.Name})
	}
	return nil
}

func (h *sqbDeploymentListHandler) deleteByLabel(label map[string]string) error {
	sqbdeploymentList := &qav1alpha1.SQBDeploymentList{}
	err := k8sclient.List(h.ctx, sqbdeploymentList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(label),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	for _, sqbdeployment := range sqbdeploymentList.Items {
		sqbdeployment.Annotations[entity.ExplicitDeleteAnnotationKey] = util.GetDeleteCheckSum(sqbdeployment.Name)
		if err = CreateOrUpdate(h.ctx, &sqbdeployment); err != nil {
			return err
		}
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
