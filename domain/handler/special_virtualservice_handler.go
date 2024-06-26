package handler

import (
	"context"
	types2 "github.com/gogo/protobuf/types"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	istioapi "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type specialVirtualServiceHandler struct {
	sqbdeployment *qav1alpha1.SQBDeployment
	ctx           context.Context
}

func NewSpecialVirtualServiceHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) *specialVirtualServiceHandler {
	return &specialVirtualServiceHandler{sqbdeployment: sqbdeployment, ctx: ctx}
}

func (h *specialVirtualServiceHandler) CreateOrUpdate() error {
	specialvirtualservice := &istio.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: specialvirtualservice.Namespace, Name: specialvirtualservice.Name}, specialvirtualservice)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	sqbapplication := &qav1alpha1.SQBApplication{}
	err = k8sclient.Get(h.ctx, client.ObjectKey{Namespace: specialvirtualservice.Namespace, Name: h.sqbdeployment.Spec.Selector.App}, sqbapplication)
	if err != nil {
		return err
	}

	virtualserviceHosts := []string{entity.ConfigMapData.GetDomainNameByClass(h.sqbdeployment.Name, SpecialVirtualServiceIngress(h.sqbdeployment))}
	specialvirtualservice.Spec.Hosts = virtualserviceHosts
	specialvirtualservice.Spec.Gateways = entity.ConfigMapData.IstioGateways()

	httproutes := make([]*istioapi.HTTPRoute, 0)

	for _, path := range sqbapplication.Spec.Subpaths {
		httpRoute := &istioapi.HTTPRoute{
			Match: []*istioapi.HTTPMatchRequest{
				{
					Uri: &istioapi.StringMatch{
						MatchType: &istioapi.StringMatch_Prefix{Prefix: path.Path},
					},
				},
			},
			Route: []*istioapi.HTTPRouteDestination{
				{Destination: &istioapi.Destination{
					Host:   path.ServiceName,
					Subset: h.sqbdeployment.Labels[entity.PlaneKey],
				}},
			},
			Headers: &istioapi.Headers{
				Request: &istioapi.Headers_HeaderOperations{Set: map[string]string{entity.XEnvFlag: h.sqbdeployment.Labels[entity.PlaneKey]}},
			},
			Timeout: &types2.Duration{Seconds: entity.ConfigMapData.IstioTimeout()},
		}
		httproutes = append(httproutes, httpRoute)
	}
	httproutes = append(httproutes, &istioapi.HTTPRoute{
		Route: []*istioapi.HTTPRouteDestination{
			{Destination: &istioapi.Destination{
				Host:   h.sqbdeployment.Spec.Selector.App,
				Subset: h.sqbdeployment.Labels[entity.PlaneKey],
			}},
		},
		Timeout: &types2.Duration{Seconds: entity.ConfigMapData.IstioTimeout()},
		Headers: &istioapi.Headers{
			Request: &istioapi.Headers_HeaderOperations{Set: map[string]string{entity.XEnvFlag: h.sqbdeployment.Labels[entity.PlaneKey]}},
		},
	})

	specialvirtualservice.Spec.Http = httproutes
	specialvirtualservice.Labels = util.MergeStringMap(specialvirtualservice.Labels, h.sqbdeployment.Labels)
	return CreateOrUpdate(h.ctx, specialvirtualservice)
}

func (h *specialVirtualServiceHandler) Delete() error {
	specialvirtualservice := &istio.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbdeployment.Namespace, Name: h.sqbdeployment.Name}}
	return Delete(h.ctx, specialvirtualservice)
}

func (h *specialVirtualServiceHandler) Handle() error {
	if !entity.ConfigMapData.IstioEnable() {
		return nil
	}
	deleted, _ := IsDeleted(h.sqbdeployment)
	if HasPublicEntry(h.sqbdeployment) && !deleted {
		return h.CreateOrUpdate()
	}
	return h.Delete()
}
