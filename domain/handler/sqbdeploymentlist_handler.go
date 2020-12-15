package handler

import (
	"context"
	"encoding/json"
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
	// 判断是否需要注入sidecar
	sqbdeployments, err := h.listByLabel(map[string]string{entity.AppKey: h.sqbapplication.Name})
	if err != nil {
		return err
	}
	sidecarInject := make(map[string]string)
	if IsIstioInject(h.sqbapplication) {
		sidecarInject[entity.IstioSidecarInjectKey] = "true"
	} else {
		sidecarInject[entity.IstioSidecarInjectKey] = "false"
	}

	for _, sqbdeployment := range sqbdeployments {
		anno := make(map[string]string)
		if podanno, ok := sqbdeployment.Annotations[entity.PodAnnotationKey]; ok {
			if err = json.Unmarshal([]byte(podanno), &anno); err != nil {
				return err
			}
			anno = util.MergeStringMap(anno, sidecarInject)
		} else {
			anno = sidecarInject
		}
		passanno, err := json.Marshal(anno)
		if err != nil {
			return err
		}
		sqbdeployment.Annotations[entity.PodAnnotationKey] = string(passanno)
		if err = CreateOrUpdate(h.ctx, &sqbdeployment); err != nil {
			return err
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
	err := k8sclient.List(h.ctx, sqbdeploymentList, &client.ListOptions{
		LabelSelector: labels.SelectorFromSet(label),
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
