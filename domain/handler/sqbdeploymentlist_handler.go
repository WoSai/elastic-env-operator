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

func (h *sqbDeploymentListHandler) CreateOrUpdateForSqbapplication() error {
	// 开启/关闭 sqbapplication的部分配置，所有对应的sqbdeployment都需要实时更新
	sqbdeployments, err := h.listByLabel(map[string]string{entity.AppKey: h.sqbapplication.Name})
	if err != nil {
		return err
	}

	for _, sqbdeployment := range sqbdeployments {
		changed := false
		// 处理istio
		h.handleIstio(&sqbdeployment, &changed)
		if changed {
			if err = CreateOrUpdate(h.ctx, &sqbdeployment); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *sqbDeploymentListHandler) DeleteForSqbapplication() error {
	return h.deleteByLabel(map[string]string{entity.AppKey: h.sqbapplication.Name})
}

func (h *sqbDeploymentListHandler) DeleteForSqbplane() error {
	return h.deleteByLabel(map[string]string{entity.PlaneKey: h.sqbplane.Name})
}

func (h *sqbDeploymentListHandler) listByLabel(label map[string]string) ([]qav1alpha1.SQBDeployment, error) {
	sqbdeploymentList := &qav1alpha1.SQBDeploymentList{}
	var ns string
	if h.sqbapplication != nil {
		ns = h.sqbapplication.Namespace
	} else if h.sqbplane != nil {
		ns = h.sqbplane.Namespace
	}
	err := k8sclient.List(h.ctx, sqbdeploymentList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(label),
		Namespace:     ns,
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}
	return sqbdeploymentList.Items, nil
}

func (h *sqbDeploymentListHandler) deleteByLabel(label map[string]string) error {
	sqbdeployments, err := h.listByLabel(label)
	if err != nil {
		return err
	}
	for _, sqbdeployment := range sqbdeployments {
		sqbdeployment.Annotations[entity.ExplicitDeleteAnnotationKey] = util.GetDeleteCheckSum(sqbdeployment.Name)
		if err = CreateOrUpdate(h.ctx, &sqbdeployment); err != nil {
			return err
		}
	}
	return nil
}

func (h *sqbDeploymentListHandler) Handle() error {
	if h.sqbapplication != nil {
		if deleted, _ := IsDeleted(h.sqbapplication); deleted {
			return h.DeleteForSqbapplication()
		}
		return h.CreateOrUpdateForSqbapplication()
	}
	if h.sqbplane != nil {
		if deleted, _ := IsDeleted(h.sqbplane); deleted {
			return h.DeleteForSqbplane()
		}
	}
	return nil
}

func (h *sqbDeploymentListHandler) handleIstio(sqbdeployment *qav1alpha1.SQBDeployment, changed *bool) {
	if sqbdeployment.Annotations[entity.InitializeAnnotationKey] != "true" {
		return
	}
	if IsIstioInject(h.sqbapplication) {
		if sqbdeployment.Annotations[entity.IstioInjectAnnotationKey] == "true" {
			return
		}
		sqbdeployment.Annotations[entity.IstioInjectAnnotationKey] = "true"
		*changed = true
	} else {
		if sqbdeployment.Annotations[entity.IstioInjectAnnotationKey] == "false" {
			return
		}
		sqbdeployment.Annotations[entity.IstioInjectAnnotationKey] = "false"
		*changed = true
	}
}
