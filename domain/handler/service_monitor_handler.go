package handler

import (
	"context"
	"encoding/json"
	prometheus "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type serviceMonitorHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx            context.Context
}

func NewServiceMonitorHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *serviceMonitorHandler {
	return &serviceMonitorHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func (h *serviceMonitorHandler) CreateOrUpdate() error {
	serviceMonitor := &prometheus.ServiceMonitor{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: serviceMonitor.Namespace, Name: serviceMonitor.Name}, serviceMonitor)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	serviceMonitor.Spec.TargetLabels = []string{entity.GroupKey}
	serviceMonitor.Spec.Selector.MatchLabels = map[string]string{
		entity.AppKey:   h.sqbapplication.Name,
		entity.GroupKey: h.sqbapplication.Labels[entity.GroupKey],
	}
	serviceMonitor.Spec.NamespaceSelector.MatchNames = []string{h.sqbapplication.Namespace}

	endpoints := make([]prometheus.Endpoint, 0)
	if err = json.Unmarshal([]byte(h.sqbapplication.Annotations[entity.ServiceMonitorAnnotationKey]), &endpoints); err != nil {
		return err
	}
	serviceMonitor.Spec.Endpoints = endpoints
	serviceMonitor.Labels = util.MergeStringMap(serviceMonitor.Labels, h.sqbapplication.Labels)
	return CreateOrUpdate(h.ctx, serviceMonitor)
}

func (h *serviceMonitorHandler) Delete() error {
	service := &prometheus.ServiceMonitor{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	return Delete(h.ctx, service)
}

func (h *serviceMonitorHandler) Handle() error {
	if !entity.ConfigMapData.IsServiceMonitorEnable() {
		return nil
	}
	if deleted, _ := IsDeleted(h.sqbapplication); deleted {
		return h.Delete()
	}

	if h.sqbapplication.Annotations[entity.ServiceMonitorAnnotationKey] != "" {
		return h.CreateOrUpdate()
	}
	return h.Delete()
}
