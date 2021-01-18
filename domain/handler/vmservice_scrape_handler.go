package handler

import (
	"context"
	"encoding/json"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/v1beta1"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type vmserviceScrapeHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx            context.Context
}

func NewVMServiceScrapeHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *vmserviceScrapeHandler {
	return &vmserviceScrapeHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func (h *vmserviceScrapeHandler) CreateOrUpdate() error {
	vmservice := &vmv1beta1.VMServiceScrape{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: vmservice.Namespace, Name: vmservice.Name}, vmservice)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	vmservice.Spec.TargetLabels = []string{entity.GroupKey}
	vmservice.Spec.Selector.MatchLabels = map[string]string{
		entity.AppKey:   h.sqbapplication.Name,
		entity.GroupKey: h.sqbapplication.Labels[entity.GroupKey],
	}
	vmservice.Spec.NamespaceSelector.MatchNames = []string{h.sqbapplication.Namespace}

	endpoints := make([]vmv1beta1.Endpoint, 0)
	if err = json.Unmarshal([]byte(h.sqbapplication.Annotations[entity.ServiceMonitorAnnotationKey]), &endpoints); err != nil {
		return err
	}
	for i, endpoint := range endpoints {
		endpoint.BearerTokenSecret.Key = ""
		endpoints[i] = endpoint
	}
	vmservice.Spec.Endpoints = endpoints
	vmservice.Labels = h.sqbapplication.Labels
	return CreateOrUpdate(h.ctx, vmservice)
}

func (h *vmserviceScrapeHandler) Delete() error {
	service := &vmv1beta1.VMServiceScrape{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	return Delete(h.ctx, service)
}

func (h *vmserviceScrapeHandler) Handle() error {
	if !entity.ConfigMapData.IsVictoriaMetricsEnable() {
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
