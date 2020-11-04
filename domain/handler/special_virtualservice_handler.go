package handler

import (
	"context"
	types2 "github.com/gogo/protobuf/types"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	istioapi "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type speccialVirtualServiceHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx context.Context
}

func newSpecialVirtualServiceHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) *speccialVirtualServiceHandler {
	return &speccialVirtualServiceHandler{sqbdeployment: sqbdeployment, ctx: ctx}
}

func (h *speccialVirtualServiceHandler) CreateOrUpdate() error {
	specialvirtualservice := &istio.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: specialvirtualservice.Namespace, Name: specialvirtualservice.Name}, specialvirtualservice)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	hosts := entity.ConfigMapData.GetDomainNames(h.sqbdeployment.Name)
	result := hosts[0]
	for _, host := range hosts[1:] {
		if len(host) < len(result) {
			result = host
		}
	}
	virtualserviceHosts := []string{result}
	specialvirtualservice.Spec.Hosts = virtualserviceHosts
	specialvirtualservice.Spec.Gateways = entity.ConfigMapData.IstioGateways()
	specialvirtualservice.Spec.Http = []*istioapi.HTTPRoute{
		{
			Route: []*istioapi.HTTPRouteDestination{
				{Destination: &istioapi.Destination{
					Host:   h.sqbdeployment.Labels[entity.AppKey],
					Subset: h.sqbdeployment.Name,
				}},
			},
			Timeout: &types2.Duration{Seconds: entity.ConfigMapData.IstioTimeout()},
			Headers: &istioapi.Headers{
				Request: &istioapi.Headers_HeaderOperations{Set: map[string]string{entity.XEnvFlag: h.sqbdeployment.Labels[entity.PlaneKey]}},
			},
		},
	}
	return CreateOrUpdate(h.ctx, specialvirtualservice)
}

func (h *speccialVirtualServiceHandler) Delete() error {
	specialvirtualservice := &istio.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, specialvirtualservice)
}