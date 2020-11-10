package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type sqbDeploymentListHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	sqbplane       *qav1alpha1.SQBPlane
	ctx            context.Context
}

func NewSqbDeploymentListHandler(sqbapplication *qav1alpha1.SQBApplication, sqbplane *qav1alpha1.SQBPlane, ctx context.Context) *deploymentListHandler {
	return &deploymentListHandler{sqbapplication: sqbapplication, sqbplane: sqbplane, ctx: ctx}
}

func (h *sqbDeploymentListHandler) DeleteSQBApplication() error {
	return k8sclient.DeleteAllOf(h.ctx, &qav1alpha1.SQBDeployment{}, &client.DeleteAllOfOptions{
		ListOptions: client.ListOptions{
			Namespace:     h.sqbapplication.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: h.sqbapplication.Name}),
		},
	})
}

func (h *sqbDeploymentListHandler) DeleteSQBPlane() error {
	return k8sclient.DeleteAllOf(h.ctx, &qav1alpha1.SQBDeployment{}, &client.DeleteAllOfOptions{
		ListOptions: client.ListOptions{
			Namespace:     h.sqbplane.Namespace,
			LabelSelector: labels.SelectorFromSet(map[string]string{entity.PlaneKey: h.sqbplane.Name}),
		},
	})
}

func (h *sqbDeploymentListHandler) Handle() error {
	if h.sqbapplication != nil && IsExplicitDelete(h.sqbapplication) {
		return h.DeleteSQBApplication()
	}
	if h.sqbplane != nil && IsExplicitDelete(h.sqbplane) {
		return h.DeleteSQBPlane()
	}
	return nil
}
