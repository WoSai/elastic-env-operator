package handler

import (
	"context"
	"encoding/json"
	vmv1beta1 "github.com/VictoriaMetrics/operator/api/v1beta1"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type vmserviceScrapeHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx            context.Context
}

type grayVmServiceScrapeHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx           context.Context
	plane         string
}

func NewVMServiceScrapeHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *vmserviceScrapeHandler {
	return &vmserviceScrapeHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func NewGrayVMServiceScrapeHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) SQBHandler {
	return &grayVmServiceScrapeHandler{sqbdeployment: sqbdeployment, ctx: ctx, plane: "gray"}
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
	// 线上配置，增加version：base
	//if entity.ConfigMapData.Env() == entity.ENV_PROD {
	//	vmservice.Spec.Selector.MatchLabels[entity.PlaneKey] = entity.ConfigMapData.BaseFlag()
	//}
	// 去掉selector的version字段
	delete(vmservice.Spec.Selector.MatchLabels, entity.PlaneKey)
	vmservice.Spec.NamespaceSelector.MatchNames = []string{h.sqbapplication.Namespace}

	endpoints := make([]vmv1beta1.Endpoint, 0)
	if err = json.Unmarshal([]byte(h.sqbapplication.Annotations[entity.ServiceMonitorAnnotationKey]), &endpoints); err != nil {
		return err
	}
	for i, endpoint := range endpoints {
		endpoint.BearerTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		endpoints[i] = endpoint
	}
	vmservice.Spec.Endpoints = endpoints
	vmservice.Labels = util.MergeStringMap(vmservice.Labels, h.sqbapplication.Labels)
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

func (h *grayVmServiceScrapeHandler) CreateOrUpdate() error {
	if entity.ConfigMapData.Env() != entity.ENV_PROD || h.sqbdeployment.Spec.Selector.Plane != h.plane {
		return nil
	}
	sqbapplication := &qav1alpha1.SQBApplication{}
	if err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Spec.Selector.App},
		sqbapplication); err != nil {
		return err
	}

	vmservice := &vmv1beta1.VMServiceScrape{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: vmservice.Namespace, Name: vmservice.Name}, vmservice)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	vmservice.Spec.TargetLabels = []string{entity.GroupKey}
	vmservice.Spec.Selector.MatchLabels = map[string]string{
		entity.AppKey:   sqbapplication.Name,
		entity.GroupKey: sqbapplication.Labels[entity.GroupKey],
		entity.PlaneKey: h.plane,
	}
	vmservice.Spec.NamespaceSelector.MatchNames = []string{h.sqbdeployment.Namespace}

	endpoints := make([]vmv1beta1.Endpoint, 0)
	if err = json.Unmarshal([]byte(sqbapplication.Annotations[entity.ServiceMonitorAnnotationKey]), &endpoints); err != nil {
		return err
	}
	for i, endpoint := range endpoints {
		endpoint.BearerTokenFile = "/var/run/secrets/kubernetes.io/serviceaccount/token"
		endpoints[i] = endpoint
	}
	vmservice.Spec.Endpoints = endpoints
	vmservice.Labels = util.MergeStringMap(vmservice.Labels, sqbapplication.Labels)
	return CreateOrUpdate(h.ctx, vmservice)
}

func (h *grayVmServiceScrapeHandler) Delete() error {
	service := &vmv1beta1.VMServiceScrape{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, service)
}

func (h *grayVmServiceScrapeHandler) Handle() error {
	if !entity.ConfigMapData.IsVictoriaMetricsEnable() {
		return nil
	}
	if deleted, _ := IsDeleted(h.sqbdeployment); deleted {
		return h.Delete()
	}
	return h.CreateOrUpdate()
}
