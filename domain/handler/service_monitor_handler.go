package handler

import (
	"context"
	prometheus "github.com/coreos/prometheus-operator/pkg/apis/monitoring/v1"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
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
	serviceMonitor.Spec.Endpoints = h.sqbapplication.Spec.Monitors

	serviceMonitor.Labels = h.sqbapplication.Labels
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
	if deleted, _ := IsDeleted(h.sqbapplication); deleted || len(h.sqbapplication.Spec.Monitors) == 0 {
		return h.Delete()
	}
	if IsServiceMonitorOpen(h.sqbapplication) {
		return h.CreateOrUpdate()
	}
	return h.Delete()
}
