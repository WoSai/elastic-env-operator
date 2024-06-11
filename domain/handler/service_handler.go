package handler

import (
	"context"
	"encoding/json"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type serviceHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx            context.Context
}

type grayServiceHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx           context.Context
	plane         string
}

func NewServiceHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *serviceHandler {
	return &serviceHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func NewGrayServiceHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) SQBHandler {
	return &grayServiceHandler{sqbdeployment: sqbdeployment, ctx: ctx, plane: "gray"}
}

func (h *serviceHandler) CreateOrUpdate() error {
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: service.Namespace, Name: service.Name}, service)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	service.Spec.Ports = h.sqbapplication.Spec.Ports
	// 兼容线上的配置，因为pod的label不能更改，所以service的selector也不能更改
	service.Spec.Selector = util.MergeStringMap(map[string]string{entity.AppKey: h.sqbapplication.Name},
		service.Spec.Selector)
	if anno, ok := h.sqbapplication.Annotations[entity.ServiceAnnotationKey]; ok {
		service.Annotations = make(map[string]string)
		_ = json.Unmarshal([]byte(anno), &service.Annotations)
	} else {
		service.Annotations = nil
	}
	service.Labels = util.MergeStringMap(service.Labels, h.sqbapplication.Labels)
	// 如果是线上配置，selector需要加上version：base， label需要加上base
	//if entity.ConfigMapData.Env() == entity.ENV_PROD {
	//	service.Spec.Selector[entity.PlaneKey] = entity.ConfigMapData.BaseFlag()
	//	service.Labels[entity.PlaneKey] = entity.ConfigMapData.BaseFlag()
	//}
	// 去掉selector的version字段
	delete(service.Spec.Selector, entity.PlaneKey)
	return CreateOrUpdate(h.ctx, service)
}

func (h *serviceHandler) Delete() error {
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	return Delete(h.ctx, service)
}

func (h *serviceHandler) Handle() error {
	if deleted, _ := IsDeleted(h.sqbapplication); deleted {
		return h.Delete()
	}
	return h.CreateOrUpdate()
}

func (h *grayServiceHandler) CreateOrUpdate() error {
	// 生产环境，对于gray的部署，需要创建gray的svc
	if entity.ConfigMapData.Env() != entity.ENV_PROD || h.sqbdeployment.Spec.Selector.Plane != h.plane {
		return nil
	}
	sqbapplication := &qav1alpha1.SQBApplication{}
	if err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Spec.Selector.App},
		sqbapplication); err != nil {
		return err
	}
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: service.Namespace, Name: service.Name}, service)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	service.Spec.Ports = sqbapplication.Spec.Ports
	service.Spec.Selector = map[string]string{
		entity.AppKey:   sqbapplication.Name,
		entity.PlaneKey: h.plane,
	}
	service.Labels = util.MergeStringMap(service.Labels, sqbapplication.Labels)
	service.Labels[entity.PlaneKey] = h.plane
	return CreateOrUpdate(h.ctx, service)
}

func (h *grayServiceHandler) Delete() error {
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, service)
}

func (h *grayServiceHandler) Handle() error {
	if deleted, _ := IsDeleted(h.sqbdeployment); deleted {
		return h.Delete()
	}
	return h.CreateOrUpdate()
}
