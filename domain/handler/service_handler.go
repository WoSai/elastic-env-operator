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

func NewServiceHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *serviceHandler {
	return &serviceHandler{sqbapplication: sqbapplication, ctx: ctx}
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
		_ = json.Unmarshal([]byte(anno), &service.Annotations)
	} else {
		service.Annotations = nil
	}
	service.Labels = util.MergeStringMap(service.Labels, h.sqbapplication.Labels)
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
