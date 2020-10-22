package entity

import (
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	types2 "github.com/gogo/protobuf/types"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/util"
	istioapi "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"strconv"
	"strings"
)

type SQBApplication struct {
	qav1alpha1.SQBApplication
	Sqbdeployment   *SQBDeployment
	Sqbplane        *SQBPlane
	Service         *corev1.Service
	Deployments     *appv1.DeploymentList
	Ingress         *v1beta1.Ingress
	Destinationrule *istio.DestinationRule
	Virtualservice  *istio.VirtualService
}

func (in *SQBApplication) BuildSelf() {
	if in.Status.Initialized {
		return
	}
	if globalDefaultDeploy, ok := ConfigMapData.GlobalDeploy(); ok {
		applicationDeploy, _ := json.Marshal(in.Spec.DeploySpec)
		applicationDeploy, _ = jsonpatch.MergePatch([]byte(globalDefaultDeploy), applicationDeploy)
		deploy := qav1alpha1.DeploySpec{}
		if err := json.Unmarshal(applicationDeploy, &deploy); err == nil {
			in.Spec.DeploySpec = deploy
		}
	}
	controllerutil.AddFinalizer(in, SqbapplicationFinalizer)
	in.Spec.Hosts = in.getIngressHosts()
	// 添加一条默认的subpath /在最后
	in.Spec.Subpaths = append(in.Spec.Subpaths, qav1alpha1.Subpath{
		Path: "/", ServiceName: in.Name, ServicePort: 80})
	in.Status.Initialized = true
}

func (in *SQBApplication) BuildRef() {
	in.buildService()
	in.buildIngress()
	in.buildVirtualService()
	in.buildDestinationRule()
	in.buildSQBDeploymentAndSQBPlane()
}

func (in *SQBApplication) buildSQBDeploymentAndSQBPlane() {
	if in.Deployments == nil {
		return
	}
	if len(in.Deployments.Items) == 0 {
		in.Sqbdeployment = &SQBDeployment{}
		in.Sqbdeployment.ObjectMeta = metav1.ObjectMeta{
			Namespace: in.Namespace,
			Name:      util.GetSubsetName(in.Name, "base"),
		}
		in.Sqbdeployment.Spec = qav1alpha1.SQBDeploymentSpec{
			Selector: qav1alpha1.Selector{
				App:   in.Name,
				Plane: "base",
			},
			DeploySpec: in.Spec.DeploySpec,
		}
		in.Sqbplane = &SQBPlane{}
		in.Sqbplane.ObjectMeta = metav1.ObjectMeta{Namespace: in.Namespace, Name: "base"}
		in.Sqbplane.Spec = qav1alpha1.SQBPlaneSpec{Description: "base environment"}
	}
}

func (in *SQBApplication) buildService() {
	if in.Service == nil {
		return
	}
	ports := make([]corev1.ServicePort, 0)
	for _, port := range in.Spec.Ports {
		port.Name = strings.ToLower(string(port.Protocol)) + "-" + strconv.Itoa(int(port.Port))
		if port.Protocol != corev1.ProtocolUDP {
			port.Protocol = corev1.ProtocolTCP
		}
		ports = append(ports, port)
	}
	in.Service.Spec.Ports = ports
	in.Service.Spec.Selector = map[string]string{
		AppKey: in.Name,
	}
	if anno, ok := in.Annotations[ServiceAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &in.Service.Annotations)
	} else {
		in.Service.Annotations = nil
	}
	in.Service.Labels = in.Labels
}

func (in *SQBApplication) buildIngress() {
	if in.Ingress == nil {
		return
	}
	rules := make([]v1beta1.IngressRule, 0)
	for _, host := range in.Spec.Hosts {
		paths := make([]v1beta1.HTTPIngressPath, 0)
		for _, subpath := range in.Spec.Subpaths {
			var path v1beta1.HTTPIngressPath
			if in.IsIstioInject() {
				path = v1beta1.HTTPIngressPath{
					Backend: v1beta1.IngressBackend{
						ServiceName: "istio-ingressgateway" + "-" + in.Namespace,
						ServicePort: intstr.FromInt(80),
					},
				}
			} else {
				path = v1beta1.HTTPIngressPath{
					Path: subpath.Path,
					Backend: v1beta1.IngressBackend{
						ServiceName: subpath.ServiceName,
						ServicePort: intstr.FromInt(subpath.ServicePort),
					},
				}
			}
			paths = append(paths, path)
		}
		rule := v1beta1.IngressRule{
			Host: host,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		}
		rules = append(rules, rule)
	}
	in.Ingress.Spec.Rules = rules
	if anno, ok := in.Annotations[IngressAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &in.Ingress.Annotations)
	} else {
		in.Ingress.Annotations = nil
	}
}

func (in *SQBApplication) buildVirtualService() {
	if in.Virtualservice == nil {
		return
	}
	virtualserviceHosts := append(in.Spec.Hosts, in.Name)
	gateways := ConfigMapData.IstioGateways()
	in.Virtualservice.Spec.Hosts = virtualserviceHosts
	in.Virtualservice.Spec.Gateways = gateways
	in.Virtualservice.Spec.Http = in.getOrGenerateHttpRoutes(in.Virtualservice.Spec.Http)
	// 处理tcp route
	for _, port := range in.Spec.Ports {
		if util.ContainString([]string{"tcp", "mongo", "mysql", "redis"}, strings.ToLower(string(port.Protocol))) {
			in.Virtualservice.Spec.Tcp = in.getOrGenerateTcpRoutes(in.Virtualservice.Spec.Tcp)
			break
		} else {
			in.Virtualservice.Spec.Tcp = nil
		}
	}
	if anno, ok := in.Annotations[VirtualServiceAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &in.Virtualservice.Annotations)
	} else {
		in.Virtualservice.Annotations = nil
	}
}

func (in *SQBApplication) buildDestinationRule() {
	if in.Destinationrule == nil {
		return
	}
	in.Destinationrule.Spec.Host = in.Name
	in.Destinationrule.Spec.Subsets = in.generateSubsets()
	if anno, ok := in.Annotations[DestinationRuleAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &in.Destinationrule.Annotations)
	} else {
		in.Destinationrule.Annotations = nil
	}
}

func (in *SQBApplication) IsIstioInject() bool {
	if ConfigMapData.IstioEnable() {
		if istioInject, ok := in.Annotations[IstioInjectAnnotationKey]; ok {
			return istioInject == "true"
		}
		return true
	}
	return false
}

func (in *SQBApplication) IsIngressOpen() bool {
	if is, ok := in.Annotations[IngressOpenAnnotationKey]; ok {
		return is == "true"
	}
	return ConfigMapData.IngressOpen()
}

func (in *SQBApplication) IsExplicitDelete() bool {
	if deleteCheckSum, ok := in.Annotations[ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == util.GetDeleteCheckSum(in.Name) {
		return true
	}
	return false
}

func (in *SQBApplication) getOrGenerateHttpRoutes(httpRoutes []*istioapi.HTTPRoute) []*istioapi.HTTPRoute {
	resultHttpRoutes := make([]*istioapi.HTTPRoute, 0)
	subpaths := in.Spec.Subpaths
	planes := in.Status.Planes
	// 特殊处理base,base需要放在最后
	_, ok := planes["base"]
	if ok {
		delete(planes, "base")
	}
	// plane+subpath决定一条route
	// 处理特性环境
	for plane := range planes {
		for _, subpath := range subpaths {
			// 查找httproute,用户有可能手动修改route,所以保留原route
			found, route := findRoute(HTTPRoutes(httpRoutes), subpath.ServiceName, plane)
			if found {
				httpRoute := istioapi.HTTPRoute(route.(HTTPRoute))
				resultHttpRoutes = append(resultHttpRoutes, &httpRoute)
				continue
			}
			// 生成httproute
			httpRoute := &istioapi.HTTPRoute{
				Route: []*istioapi.HTTPRouteDestination{
					{Destination: &istioapi.Destination{
						Host:   subpath.ServiceName,
						Subset: util.GetSubsetName(subpath.ServiceName, plane),
					}},
				},
				Timeout: &types2.Duration{Seconds: ConfigMapData.IstioTimeout()},
			}
			headerMatchRequest := &istioapi.HTTPMatchRequest{}
			queryparamsMatchRequest := &istioapi.HTTPMatchRequest{}
			sourcelabelsMatchRequest := &istioapi.HTTPMatchRequest{}
			uriStringMatch := &istioapi.StringMatch{
				MatchType: &istioapi.StringMatch_Prefix{Prefix: subpath.Path},
			}
			envFlagMatchMap := map[string]*istioapi.StringMatch{
				XEnvFlag: {MatchType: &istioapi.StringMatch_Exact{Exact: plane}},
			}
			headerMatchRequest.Headers = envFlagMatchMap
			queryparamsMatchRequest.QueryParams = envFlagMatchMap
			sourcelabelsMatchRequest.SourceLabels = map[string]string{PlaneKey: plane}
			if subpath.Path != "/" {
				headerMatchRequest.Uri = uriStringMatch
				queryparamsMatchRequest.Uri = uriStringMatch
				sourcelabelsMatchRequest.Uri = uriStringMatch
			}
			httpRoute.Match = []*istioapi.HTTPMatchRequest{headerMatchRequest, queryparamsMatchRequest, sourcelabelsMatchRequest}
			httpRoute.Headers = &istioapi.Headers{
				Request: &istioapi.Headers_HeaderOperations{Set: map[string]string{XEnvFlag: plane}},
			}
			resultHttpRoutes = append(resultHttpRoutes, httpRoute)
		}
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
			httpRoute := &istioapi.HTTPRoute{
				Route: []*istioapi.HTTPRouteDestination{
					{Destination: &istioapi.Destination{
						Host:   subpath.ServiceName,
						Subset: util.GetSubsetName(subpath.ServiceName, "base"),
					}},
				},
				Timeout: &types2.Duration{Seconds: ConfigMapData.IstioTimeout()},
			}
			if subpath.Path != "/" {
				httpRoute.Match = []*istioapi.HTTPMatchRequest{
					{
						Uri: &istioapi.StringMatch{
							MatchType: &istioapi.StringMatch_Prefix{Prefix: subpath.Path},
						},
					},
				}
			}
			resultHttpRoutes = append(resultHttpRoutes, httpRoute)
		}
	}
	return resultHttpRoutes
}

// 根据plane生成DestinationRule的subsets
func (in *SQBApplication) generateSubsets() []*istioapi.Subset {
	subsets := make([]*istioapi.Subset, 0)
	for plane := range in.Status.Planes {
		subsets = append(subsets, &istioapi.Subset{
			Name: util.GetSubsetName(in.Name, plane),
			Labels: map[string]string{
				PlaneKey: plane,
			},
		})
	}
	return subsets
}

// 根据plane生成tcp route
func (in *SQBApplication) getOrGenerateTcpRoutes(tcpRoutes []*istioapi.TCPRoute) []*istioapi.TCPRoute {
	resultTcpRoutes := make([]*istioapi.TCPRoute, 0)
	planes := in.Status.Planes
	_, ok := planes["base"]
	if ok {
		delete(planes, "base")
	}
	// 处理特性环境
	for plane := range planes {
		// 查找匹配的tcproute
		found, route := findRoute(TCPRoutes(tcpRoutes), in.Name, plane)
		if found {
			tcpRoute := istioapi.TCPRoute(route.(TCPRoute))
			resultTcpRoutes = append(resultTcpRoutes, &tcpRoute)
			continue
		}
		// 生成tcproute
		tcpRoute := &istioapi.TCPRoute{
			Route: []*istioapi.RouteDestination{
				{Destination: &istioapi.Destination{
					Host:   in.Name,
					Subset: util.GetSubsetName(in.Name, plane),
				}},
			},
			Match: []*istioapi.L4MatchAttributes{
				{SourceLabels: map[string]string{
					PlaneKey: plane,
				}},
			},
		}
		resultTcpRoutes = append(resultTcpRoutes, tcpRoute)
	}
	// 处理基础环境
	if ok {
		planes["base"] = 1
		found, route := findRoute(TCPRoutes(tcpRoutes), in.Name, "base")
		if found {
			tcpRoute := istioapi.TCPRoute(route.(TCPRoute))
			resultTcpRoutes = append(resultTcpRoutes, &tcpRoute)
		} else {
			tcpRoute := &istioapi.TCPRoute{
				Route: []*istioapi.RouteDestination{
					{Destination: &istioapi.Destination{
						Host:   in.Name,
						Subset: util.GetSubsetName(in.Name, "base"),
					}},
				},
			}
			resultTcpRoutes = append(resultTcpRoutes, tcpRoute)
		}
	}
	return resultTcpRoutes
}

func (in *SQBApplication) getIngressHosts() []string {
	hosts := ConfigMapData.GetDomainNames(in.Name)
	for _, host := range in.Spec.Hosts {
		if !util.ContainString(hosts, host) {
			hosts = append(hosts, host)
		}
	}
	return hosts
}

func (in *SQBApplication) getSpecialVirtualServiceHost(plane string) []string {
	hosts := ConfigMapData.GetDomainNames(util.GetSubsetName(in.Name, plane))
	result := hosts[0]
	for _, host := range hosts[1:] {
		if len(host) < len(result) {
			result = host
		}
	}
	return []string{result}
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
