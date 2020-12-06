package handler

import (
	"context"
	"encoding/json"
	types2 "github.com/gogo/protobuf/types"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	istioapi "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type virtualServiceHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx            context.Context
}

func NewVirtualServiceHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *virtualServiceHandler {
	return &virtualServiceHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func (h *virtualServiceHandler) CreateOrUpdate() error {
	virtualservice := &istio.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: virtualservice.Namespace, Name: virtualservice.Name}, virtualservice)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	virtualserviceHosts := getIngressHosts(h.sqbapplication)
	virtualserviceHosts = append(virtualserviceHosts, h.sqbapplication.Name)
	gateways := entity.ConfigMapData.IstioGateways()
	virtualservice.Spec.Hosts = virtualserviceHosts
	virtualservice.Spec.Gateways = gateways
	virtualservice.Spec.Http = h.getOrGenerateHttpRoutes(virtualservice.Spec.Http)
	// 处理tcp route
	for _, port := range h.sqbapplication.Spec.Ports {
		if util.ContainString([]string{"tcp", "mongo", "mysql", "redis"}, strings.ToLower(strings.Split(port.Name, "-")[0])) {
			virtualservice.Spec.Tcp = h.getOrGenerateTcpRoutes(virtualservice.Spec.Tcp)
			break
		} else {
			virtualservice.Spec.Tcp = nil
		}
	}
	if anno, ok := h.sqbapplication.Annotations[entity.VirtualServiceAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &virtualservice.Annotations)
	} else {
		virtualservice.Annotations = nil
	}
	virtualservice.Labels = h.sqbapplication.Labels
	return CreateOrUpdate(h.ctx, virtualservice)
}

func (h *virtualServiceHandler) Delete() error {
	virtualservice := &istio.VirtualService{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	return Delete(h.ctx, virtualservice)
}

func (h *virtualServiceHandler) Handle() error {
	if !entity.ConfigMapData.IstioEnable() {
		return nil
	}
	if deleted, _ := IsDeleted(h.sqbapplication); deleted {
		return h.Delete()
	}
	if IsIstioInject(h.sqbapplication) {
		return h.CreateOrUpdate()
	}
	return h.Delete()
}

func (h *virtualServiceHandler) getOrGenerateHttpRoutes(httpRoutes []*istioapi.HTTPRoute) []*istioapi.HTTPRoute {
	resultHttpRoutes := make([]*istioapi.HTTPRoute, 0)
	subpaths := h.sqbapplication.Spec.Subpaths
	planes := h.sqbapplication.Status.Planes
	// 特殊处理base,base需要放在最后
	_, ok := planes["base"]
	if ok {
		delete(planes, "base")
	}
	// plane+subpath决定一条route,path有顺序要求
	// 处理特性环境
	for plane := range planes {
		// 处理subpath
		for _, subpath := range subpaths {
			// 查找httproute,用户有可能手动修改route,所以保留原route
			found, route := findRoute(HTTPRoutes(httpRoutes), subpath.ServiceName, plane)
			if found {
				httpRoute := istioapi.HTTPRoute(route.(HTTPRoute))
				resultHttpRoutes = append(resultHttpRoutes, &httpRoute)
				continue
			}
			// 生成httproute
			httpRoute := generatePlaneHttpRoute(subpath.ServiceName, plane, subpath.Path)
			resultHttpRoutes = append(resultHttpRoutes, httpRoute)
		}
		// 处理默认路径
		httpRoute := generatePlaneHttpRoute(h.sqbapplication.Name, plane, "/")
		resultHttpRoutes = append(resultHttpRoutes, httpRoute)
	}
	// 处理基础环境
	if ok {
		planes["base"] = 1
		for _, subpath := range subpaths {
			found, route := findRoute(HTTPRoutes(httpRoutes), subpath.ServiceName, "base")
			if found {
				httpRoute := istioapi.HTTPRoute(route.(HTTPRoute))
				resultHttpRoutes = append(resultHttpRoutes, &httpRoute)
				continue
			}
			httpRoute := generateBaseHttpRoute(subpath.ServiceName, subpath.Path)
			resultHttpRoutes = append(resultHttpRoutes, httpRoute)
		}
		httpRoute := generateBaseHttpRoute(h.sqbapplication.Name, "/")
		resultHttpRoutes = append(resultHttpRoutes, httpRoute)
	}
	return resultHttpRoutes
}

func (h *virtualServiceHandler) getOrGenerateTcpRoutes(tcpRoutes []*istioapi.TCPRoute) []*istioapi.TCPRoute {
	resultTcpRoutes := make([]*istioapi.TCPRoute, 0)
	planes := h.sqbapplication.Status.Planes
	_, ok := planes["base"]
	if ok {
		delete(planes, "base")
	}
	// 处理特性环境
	for plane := range planes {
		// 查找匹配的tcproute
		found, route := findRoute(TCPRoutes(tcpRoutes), h.sqbapplication.Name, plane)
		if found {
			tcpRoute := istioapi.TCPRoute(route.(TCPRoute))
			resultTcpRoutes = append(resultTcpRoutes, &tcpRoute)
			continue
		}
		// 生成tcproute
		tcpRoute := &istioapi.TCPRoute{
			Route: []*istioapi.RouteDestination{
				{Destination: &istioapi.Destination{
					Host:   h.sqbapplication.Name,
					Subset: util.GetSubsetName(h.sqbapplication.Name, plane),
				}},
			},
			Match: []*istioapi.L4MatchAttributes{
				{SourceLabels: map[string]string{
					entity.PlaneKey: plane,
				}},
			},
		}
		resultTcpRoutes = append(resultTcpRoutes, tcpRoute)
	}
	// 处理基础环境
	if ok {
		planes["base"] = 1
		found, route := findRoute(TCPRoutes(tcpRoutes), h.sqbapplication.Name, "base")
		if found {
			tcpRoute := istioapi.TCPRoute(route.(TCPRoute))
			resultTcpRoutes = append(resultTcpRoutes, &tcpRoute)
		} else {
			tcpRoute := &istioapi.TCPRoute{
				Route: []*istioapi.RouteDestination{
					{Destination: &istioapi.Destination{
						Host:   h.sqbapplication.Name,
						Subset: util.GetSubsetName(h.sqbapplication.Name, "base"),
					}},
				},
			}
			resultTcpRoutes = append(resultTcpRoutes, tcpRoute)
		}
	}
	return resultTcpRoutes
}

func generatePlaneHttpRoute(host, plane, path string) *istioapi.HTTPRoute {
	httpRoute := &istioapi.HTTPRoute{
		Route: []*istioapi.HTTPRouteDestination{
			{Destination: &istioapi.Destination{
				Host:   host,
				Subset: util.GetSubsetName(host, plane),
			}},
		},
		Timeout: &types2.Duration{Seconds: entity.ConfigMapData.IstioTimeout()},
	}
	prefixMatch := &istioapi.StringMatch{
		MatchType: &istioapi.StringMatch_Prefix{Prefix: path},
	}
	exactMatch := &istioapi.StringMatch{
		MatchType: &istioapi.StringMatch_Exact{Exact: plane},
	}
	headerMatchRequest := &istioapi.HTTPMatchRequest{}
	queryparamsMatchRequest := &istioapi.HTTPMatchRequest{}
	sourcelabelsMatchRequest := &istioapi.HTTPMatchRequest{}

	headerMatchRequest.Headers = map[string]*istioapi.StringMatch{
		entity.XEnvFlag: exactMatch,
	}
	queryparamsMatchRequest.QueryParams = map[string]*istioapi.StringMatch{
		entity.XEnvFlag: exactMatch,
	}
	sourcelabelsMatchRequest.SourceLabels = map[string]string{entity.PlaneKey: plane}

	if path != "/" {
		headerMatchRequest.Uri = prefixMatch
		queryparamsMatchRequest.Uri = prefixMatch
		sourcelabelsMatchRequest.Uri = prefixMatch
	}
	httpRoute.Match = []*istioapi.HTTPMatchRequest{headerMatchRequest, queryparamsMatchRequest, sourcelabelsMatchRequest}
	httpRoute.Headers = &istioapi.Headers{
		Request: &istioapi.Headers_HeaderOperations{Set: map[string]string{entity.XEnvFlag: plane}},
	}
	return httpRoute
}

func generateBaseHttpRoute(host, path string) *istioapi.HTTPRoute {
	plane := "base"
	httpRoute := &istioapi.HTTPRoute{
		Route: []*istioapi.HTTPRouteDestination{
			{Destination: &istioapi.Destination{
				Host:   host,
				Subset: util.GetSubsetName(host, plane),
			}},
		},
		Timeout: &types2.Duration{Seconds: entity.ConfigMapData.IstioTimeout()},
	}
	if path != "/" {
		httpRoute.Match = []*istioapi.HTTPMatchRequest{
			{
				Uri: &istioapi.StringMatch{
					MatchType: &istioapi.StringMatch_Prefix{Prefix: path},
				},
			},
		}
	}
	return httpRoute
}

func getIngressHosts(sqbapplication *qav1alpha1.SQBApplication) []string {
	hosts := make([]string, 0)
	for _, domain := range sqbapplication.Spec.Domains {
		hosts  = append(hosts, domain.Host)
	}
	return hosts
}

// 辅助查找route的逻辑
type TCPRoute istioapi.TCPRoute
type HTTPRoute istioapi.HTTPRoute
type TCPRoutes []*istioapi.TCPRoute
type HTTPRoutes []*istioapi.HTTPRoute
type Route interface {
	GetHostAndSubset() (string, string)
}
type Routes interface {
	FindRoute(string, string) (bool, Route)
}

func (tcproutes TCPRoutes) FindRoute(host, plane string) (bool, Route) {
	for _, route := range tcproutes {
		found := matchRoute(TCPRoute(*route), host, plane)
		if found {
			return found, TCPRoute(*route)
		}
	}
	return false, nil
}

func (httproutes HTTPRoutes) FindRoute(host, plane string) (bool, Route) {
	for _, route := range httproutes {
		found := matchRoute(HTTPRoute(*route), host, plane)
		if found {
			return found, HTTPRoute(*route)
		}
	}
	return false, nil
}

func (t TCPRoute) GetHostAndSubset() (string, string) {
	if len(t.Route) != 0 {
		destination := t.Route[0].Destination
		return destination.Host, destination.Subset
	}
	return "", ""
}

func (h HTTPRoute) GetHostAndSubset() (string, string) {
	if len(h.Route) != 0 {
		destination := h.Route[0].Destination
		return destination.Host, destination.Subset
	}
	return "", ""
}

func matchRoute(r Route, host, plane string) bool {
	rhost, subset := r.GetHostAndSubset()
	if rhost == host && subset == util.GetSubsetName(host, plane) {
		return true
	}
	return false
}

func findRoute(rs Routes, host, plane string) (bool, Route) {
	return rs.FindRoute(host, plane)
}
