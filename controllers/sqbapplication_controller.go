/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"encoding/json"
	"github.com/go-logr/logr"
	types2 "github.com/gogo/protobuf/types"
	v1beta14 "istio.io/api/networking/v1beta1"
	v1beta13 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v12 "k8s.io/api/apps/v1"
	v13 "k8s.io/api/core/v1"
	v1beta12 "k8s.io/api/extensions/v1beta1"
	v14 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"strconv"
	"strings"

	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
)

// SQBApplicationReconciler reconciles a SQBApplication object
type SQBApplicationReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbapplications/status,verbs=get;update;patch

func (r *SQBApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	instance := &qav1alpha1.SQBApplication{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}
	return HandleReconcile(r, ctx, instance)
}

func (r *SQBApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qav1alpha1.SQBApplication{}, builder.WithPredicates(GenerationAnnotationPredicate)).
		Watches(&source.Kind{Type: &v12.Deployment{}},
			&handler.EnqueueRequestForOwner{OwnerType: &qav1alpha1.SQBApplication{}, IsController: false},
			builder.WithPredicates(CreateDeleteAnnotationPredicate)).
		Complete(r)
}

func (r *SQBApplicationReconciler) IsInitialized(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBApplication)
	if cr.Status.Initialized == true {
		return true, nil
	}
	// 设置finalizer
	controllerutil.AddFinalizer(cr, SqbapplicationFinalizer)
	err := r.Update(ctx, cr)
	if err != nil {
		return false, err
	}
	// 更新status
	cr.Status.Initialized = true
	return false, r.Status().Update(ctx, cr)
}

func (r *SQBApplicationReconciler) IsDeleting(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBApplication)
	if cr.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(cr, SqbapplicationFinalizer) {
		return false, nil
	}

	var err error

	configMapData := getDefaultConfigMapData(r.Client, ctx)

	// 如果configmap没有配置密码，直接删除资源
	password, ok := configMapData["deletePassword"]
	if !ok {
		return true, r.RemoveFinalizer(ctx, cr)
	}
	if cr.Annotations[ExplicitDeleteAnnotationKey] == "true" && cr.Annotations[DeletePasswordAnnotationKey] == password {
		// 删除ingress,service
		ingress := &v1beta12.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
		err = r.Delete(ctx, ingress)
		if err != nil && !apierrors.IsNotFound(err) {
			return true, err
		}
		service := &v13.Service{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
		err = r.Delete(ctx, service)
		if err != nil && !apierrors.IsNotFound(err) {
			return true, err
		}
		if isIstioEnable(r.Client, ctx, configMapData, cr) {
			// 如果有istio,删除virtualservice,destinationrule
			destinationrule := &v1beta13.DestinationRule{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
			err = r.Delete(ctx, destinationrule)
			if ignoreNoMatchError(err) != nil {
				return true, err
			}
			virtualservice := &v1beta13.VirtualService{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
			err = r.Delete(ctx, virtualservice)
			if ignoreNoMatchError(err) != nil {
				return true, err
			}
		}
		// 删除SQBDeployment和Deployment
		err = deleteSqbdeploymentByLabel(r.Client, ctx, cr.Namespace, map[string]string{AppKey: cr.Name})
		if err != nil {
			return true, err
		}
		// deployment会触发事件，所以最后删除
		err = deleteDeploymentByLabel(r.Client, ctx, cr.Namespace, map[string]string{AppKey: cr.Name})
		if err != nil {
			return true, err
		}
	}
	return true, r.RemoveFinalizer(ctx, cr)
}

func (r *SQBApplicationReconciler) Operate(ctx context.Context, obj runtime.Object) error {
	cr := obj.(*qav1alpha1.SQBApplication)
	var err error
	// 判断是否有对应deployment，如果没有就返回不操作
	deploymentList := &v12.DeploymentList{}
	err = r.List(ctx, deploymentList, &client.ListOptions{Namespace: cr.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{AppKey: cr.Name})})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	mirrors := map[string]string{}
	planes := map[string]string{}
	for _, deployment := range deploymentList.Items {
		mirrors[deployment.Name] = ""
		if plane, ok := deployment.Labels[PlaneKey]; ok {
			planes[plane] = ""
		}
	}
	cr.Status.Mirrors = mirrors
	cr.Status.Planes = planes
	if len(planes) == 0 {
		if cr.Spec.Replicas != nil && cr.Spec.Image != "" {
			// 如果没有环境，创建一个base环境
			basePlane := &qav1alpha1.SQBPlane{
				ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: "base"},
				Spec:       qav1alpha1.SQBPlaneSpec{Description: "base environment"},
			}
			_, err := controllerutil.CreateOrUpdate(ctx, r.Client, basePlane, func() error { return nil })
			if err != nil {
				return err
			}
			sqbDeployment := &qav1alpha1.SQBDeployment{
				ObjectMeta: v1.ObjectMeta{
					Namespace: cr.Namespace,
					Name:      getSubsetName(cr.Name, "base"),
				},
				Spec: qav1alpha1.SQBDeploymentSpec{
					Selector: qav1alpha1.Selector{
						App:   cr.Name,
						Plane: "base",
					},
					DeploySpec: cr.Spec.DeploySpec,
				},
			}
			return r.Create(ctx, sqbDeployment)
		}else{
			return nil
		}
	}
	configMapData := getDefaultConfigMapData(r.Client, ctx)
	//　处理service
	service := &v13.Service{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
		ports := make([]v13.ServicePort, 0)
		for _, port := range cr.Spec.Ports {
			port.Name = strings.ToLower(string(port.Protocol)) + "-" + strconv.Itoa(int(port.Port))
			if strings.ToUpper(string(port.Protocol)) != string(v13.ProtocolUDP) {
				port.Protocol = v13.ProtocolTCP
			}
			ports = append(ports, port)
		}
		service.Spec.Ports = ports
		service.Spec.Selector = map[string]string{
			AppKey: cr.Name,
		}
		service.Spec.Type = cr.Spec.ServiceType
		if anno, ok := cr.Annotations[ServiceAnnotationKey]; ok {
			err = json.Unmarshal([]byte(anno), &service.Annotations)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// 判断是否启用istio
	isIstioEnable := isIstioEnable(r.Client, ctx, configMapData, cr)
	// 添加一条默认的subpath /在最后
	cr.Spec.Subpaths = append(cr.Spec.Subpaths, qav1alpha1.Subpath{
		Path: "/", ServiceName: cr.Name, ServicePort: int(cr.Spec.Ports[0].Port)})
	// 处理istio相关配置
	if isIstioEnable {
		err := r.handleIstio(ctx, cr, configMapData, deploymentList)
		if err != nil {
			return err
		}
	} else {
		err := r.handleNoIstio(ctx, cr, configMapData)
		if err != nil {
			return err
		}
	}
	// 更新状态
	cr.Status.ErrorInfo = ""
	return r.Status().Update(ctx, cr)
}

func (r *SQBApplicationReconciler) ReconcileFail(ctx context.Context, obj runtime.Object, err error) {
	cr := obj.(*qav1alpha1.SQBApplication)
	cr.Status.ErrorInfo = err.Error()
	_ = r.Status().Update(ctx, cr)
}

func (r *SQBApplicationReconciler) RemoveFinalizer(ctx context.Context, cr *qav1alpha1.SQBApplication) error {
	controllerutil.RemoveFinalizer(cr, SqbapplicationFinalizer)
	return r.Update(ctx, cr)
}

// 处理启用istio的逻辑
func (r *SQBApplicationReconciler) handleIstio(ctx context.Context, cr *qav1alpha1.SQBApplication,
	configMapData map[string]string, deploymentList *v12.DeploymentList) error {
	isIngressOpen := isIngressOpen(configMapData, cr)
	hosts := getIngressHosts(configMapData, cr)
	subpaths := cr.Spec.Subpaths
	// Ingress
	ingress := &v1beta12.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	if isIngressOpen {
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			rules := make([]v1beta12.IngressRule, 0)
			ingressRule := v1beta12.IngressRuleValue{
				HTTP: &v1beta12.HTTPIngressRuleValue{
					Paths: []v1beta12.HTTPIngressPath{
						{
							Backend: v1beta12.IngressBackend{
								ServiceName: "istio-ingressgateway" + "-" + cr.Namespace,
								ServicePort: intstr.FromInt(80),
							},
						},
					},
				},
			}
			for _, host := range hosts {
				rule := v1beta12.IngressRule{
					Host: host,
					IngressRuleValue: ingressRule,
				}
				rules = append(rules, rule)
			}
			for _,deployment := range deploymentList.Items {
				if publicEntry,ok := deployment.Annotations[PublicEntryAnnotationKey]; ok && publicEntry == "true" {
					rule := v1beta12.IngressRule{
						Host: getSpecialVirtualServiceHost(configMapData, &deployment),
						IngressRuleValue: ingressRule,
					}
					rules = append(rules, rule)
				}
			}
			ingress.Spec = v1beta12.IngressSpec{Rules: rules}
			if anno, ok := cr.Annotations[IngressAnnotationKey]; ok {
				err := json.Unmarshal([]byte(anno), &ingress.Annotations)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		err := r.Delete(ctx, ingress)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}
	// DestinationRule
	planes := cr.Status.Planes
	destinationrule := &v1beta13.DestinationRule{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, destinationrule, func() error {
		destinationrule.Spec.Host = cr.Name
		destinationrule.Spec.Subsets = generateSubsets(cr, planes)
		if anno, ok := cr.Annotations[DestinationRuleAnnotationKey]; ok {
			err := json.Unmarshal([]byte(anno), &destinationrule.Annotations)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// VirtualService
	virtualservice := &v1beta13.VirtualService{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	_, err = controllerutil.CreateOrUpdate(ctx, r.Client, virtualservice, func() error {
		virtualserviceHosts := append(hosts, cr.Name)
		gateways := getIstioGateways(configMapData)
		virtualservice.Spec.Hosts = virtualserviceHosts
		virtualservice.Spec.Gateways = gateways
		virtualservice.Spec.Http = getOrGenerateHttpRoutes(virtualservice.Spec.Http, subpaths, planes, configMapData)
		// 处理tcp route
		for _, port := range cr.Spec.Ports {
			if containString([]string{"tcp", "mongo", "mysql", "redis"}, strings.ToLower(string(port.Protocol))) {
				virtualservice.Spec.Tcp = getOrGenerateTcpRoutes(virtualservice.Spec.Tcp, cr, planes)
				break
			} else {
				virtualservice.Spec.Tcp = nil
			}
		}
		if anno, ok := cr.Annotations[VirtualServiceAnnotationKey]; ok {
			err = json.Unmarshal([]byte(anno), &virtualservice.Annotations)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// 处理没有istio的逻辑
func (r *SQBApplicationReconciler) handleNoIstio(ctx context.Context, cr *qav1alpha1.SQBApplication,
	configMapData map[string]string) error {
	isIngressOpen := isIngressOpen(configMapData, cr)
	hosts := getIngressHosts(configMapData, cr)
	subpaths := cr.Spec.Subpaths
	// Ingress
	ingress := &v1beta12.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	if isIngressOpen {
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			rules := make([]v1beta12.IngressRule, 0)
			for _, host := range hosts {
				paths := make([]v1beta12.HTTPIngressPath, 0)
				for _, subpath := range subpaths {
					path := v1beta12.HTTPIngressPath{
						Path: subpath.Path,
						Backend: v1beta12.IngressBackend{
							ServiceName: subpath.ServiceName,
							ServicePort: intstr.FromInt(subpath.ServicePort),
						},
					}
					paths = append(paths, path)
				}
				rule := v1beta12.IngressRule{
					Host: host,
					IngressRuleValue: v1beta12.IngressRuleValue{
						HTTP: &v1beta12.HTTPIngressRuleValue{
							Paths: paths,
						},
					},
				}
				rules = append(rules, rule)
			}
			ingress.Spec = v1beta12.IngressSpec{Rules: rules}
			if anno, ok := cr.Annotations[IngressAnnotationKey]; ok {
				err := json.Unmarshal([]byte(anno), &ingress.Annotations)
				if err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		err := r.Delete(ctx, ingress)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
	}

	// 删除virtualservice和destinationrule
	virtualservice := &v1beta13.VirtualService{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	err := r.Delete(ctx, virtualservice)
	if ignoreNoMatchError(err) != nil {
		return err
	}

	destinationrule := &v1beta13.DestinationRule{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	err = r.Delete(ctx, destinationrule)
	if ignoreNoMatchError(err) != nil {
		return err
	}
	return nil
}

// 根据plane生成DestinationRule的subsets
func generateSubsets(cr *qav1alpha1.SQBApplication, planes map[string]string) []*v1beta14.Subset {
	subsets := make([]*v1beta14.Subset, 0)
	for plane := range planes {
		subsets = append(subsets, &v1beta14.Subset{
			Name: getSubsetName(cr.Name, plane),
			Labels: map[string]string{
				PlaneKey: plane,
			},
		})
	}
	return subsets
}

// 根据plane和subpath获取匹配的route，或者生成route
func getOrGenerateHttpRoutes(httpRoutes []*v1beta14.HTTPRoute, subpaths []qav1alpha1.Subpath, planes map[string]string,
	configMapData map[string]string) []*v1beta14.HTTPRoute {
	resultHttpRoutes := make([]*v1beta14.HTTPRoute, 0)
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
				httpRoute := v1beta14.HTTPRoute(route.(HTTPRoute))
				resultHttpRoutes = append(resultHttpRoutes, &httpRoute)
				continue
			}
			// 生成httproute
			httpRoute := &v1beta14.HTTPRoute{
				Route: []*v1beta14.HTTPRouteDestination{
					{Destination: &v1beta14.Destination{
						Host:   subpath.ServiceName,
						Subset: getSubsetName(subpath.ServiceName, plane),
					}},
				},
				Timeout: &types2.Duration{Seconds: getIstioTimeout(configMapData)},
			}
			headerMatchRequest := &v1beta14.HTTPMatchRequest{}
			queryparamsMatchRequest := &v1beta14.HTTPMatchRequest{}
			sourcelabelsMatchRequest := &v1beta14.HTTPMatchRequest{}
			uriStringMatch := &v1beta14.StringMatch{
				MatchType: &v1beta14.StringMatch_Prefix{Prefix: subpath.Path},
			}
			envFlagMatchMap := map[string]*v1beta14.StringMatch{
				XEnvFlag: {MatchType: &v1beta14.StringMatch_Exact{Exact: plane}},
			}
			headerMatchRequest.Headers = envFlagMatchMap
			queryparamsMatchRequest.QueryParams = envFlagMatchMap
			sourcelabelsMatchRequest.SourceLabels = map[string]string{PlaneKey: plane}
			if subpath.Path != "/" {
				headerMatchRequest.Uri = uriStringMatch
				queryparamsMatchRequest.Uri = uriStringMatch
				sourcelabelsMatchRequest.Uri = uriStringMatch
			}
			httpRoute.Match = []*v1beta14.HTTPMatchRequest{headerMatchRequest, queryparamsMatchRequest, sourcelabelsMatchRequest}
			httpRoute.Headers = &v1beta14.Headers{
				Request: &v1beta14.Headers_HeaderOperations{Set: map[string]string{XEnvFlag: plane}},
			}
			resultHttpRoutes = append(resultHttpRoutes, httpRoute)
		}
	}
	// 处理基础环境
	if ok {
		planes["base"] = ""
		for _, subpath := range subpaths {
			found, route := findRoute(HTTPRoutes(httpRoutes), subpath.ServiceName, "base")
			if found {
				httpRoute := v1beta14.HTTPRoute(route.(HTTPRoute))
				resultHttpRoutes = append(resultHttpRoutes, &httpRoute)
				continue
			}
			httpRoute := &v1beta14.HTTPRoute{
				Route: []*v1beta14.HTTPRouteDestination{
					{Destination: &v1beta14.Destination{
						Host:   subpath.ServiceName,
						Subset: getSubsetName(subpath.ServiceName, "base"),
					}},
				},
				Timeout: &types2.Duration{Seconds: getIstioTimeout(configMapData)},
			}
			if subpath.Path != "/" {
				httpRoute.Match = []*v1beta14.HTTPMatchRequest{
					{
						Uri: &v1beta14.StringMatch{
							MatchType: &v1beta14.StringMatch_Prefix{Prefix: subpath.Path},
						},
					},
				}
			}
			resultHttpRoutes = append(resultHttpRoutes, httpRoute)
		}
	}
	return resultHttpRoutes
}

// 根据plane生成tcp route
func getOrGenerateTcpRoutes(tcpRoutes []*v1beta14.TCPRoute, cr *qav1alpha1.SQBApplication,
	planes map[string]string) []*v1beta14.TCPRoute {
	resultTcpRoutes := make([]*v1beta14.TCPRoute, 0)
	_, ok := planes["base"]
	if ok {
		delete(planes, "base")
	}
	// 处理特性环境
	for plane := range planes {
		// 查找匹配的tcproute
		found, route := findRoute(TCPRoutes(tcpRoutes), cr.Name, plane)
		if found {
			tcpRoute := v1beta14.TCPRoute(route.(TCPRoute))
			resultTcpRoutes = append(resultTcpRoutes, &tcpRoute)
			continue
		}
		// 生成tcproute
		tcpRoute := &v1beta14.TCPRoute{
			Route: []*v1beta14.RouteDestination{
				{Destination: &v1beta14.Destination{
					Host:   cr.Name,
					Subset: getSubsetName(cr.Name, plane),
				}},
			},
			Match: []*v1beta14.L4MatchAttributes{
				{SourceLabels: map[string]string{
					PlaneKey: plane,
				}},
			},
		}
		resultTcpRoutes = append(resultTcpRoutes, tcpRoute)
	}
	// 处理基础环境
	if ok {
		planes["base"] = ""
		found, route := findRoute(TCPRoutes(tcpRoutes), cr.Name, "base")
		if found {
			tcpRoute := v1beta14.TCPRoute(route.(TCPRoute))
			resultTcpRoutes = append(resultTcpRoutes, &tcpRoute)
		} else {
			tcpRoute := &v1beta14.TCPRoute{
				Route: []*v1beta14.RouteDestination{
					{Destination: &v1beta14.Destination{
						Host:   cr.Name,
						Subset: getSubsetName(cr.Name, "base"),
					}},
				},
			}
			resultTcpRoutes = append(resultTcpRoutes, tcpRoute)
		}
	}
	return resultTcpRoutes
}

func getIstioTimeout(configMapData map[string]string) int64 {
	timeout, ok := configMapData["istioTimeout"]
	if !ok {
		timeout = "90"
	}
	routeTimeout, err := strconv.Atoi(timeout)
	if err != nil {
		routeTimeout = 90
	}
	return int64(routeTimeout)
}

type TCPRoute v1beta14.TCPRoute
type HTTPRoute v1beta14.HTTPRoute
type TCPRoutes []*v1beta14.TCPRoute
type HTTPRoutes []*v1beta14.HTTPRoute
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
	if rhost == host && subset == getSubsetName(host, plane) {
		return true
	}
	return false
}

func findRoute(rs Routes, host, plane string) (bool, Route) {
	return rs.FindRoute(host, plane)
}

func getSubsetName(host, plane string) string {
	return host + "-" + plane
}

func getIstioGateways(configMapData map[string]string) []string {
	if gateways, ok := configMapData["istioGateways"]; ok {
		return strings.Split(gateways, ",")
	}
	return []string{"mesh"}
}

func isIstioEnable(client client.Client, ctx context.Context,
	configMapData map[string]string, cr *qav1alpha1.SQBApplication) bool {
	enable := false
	var err error
	istio := &v14.CustomResourceDefinition{}
	err = client.Get(ctx, types.NamespacedName{Namespace: "", Name: "virtualservices.networking.istio.io"}, istio)
	// err==nil 表示集群安装了istio
	if err == nil {
		// 判断application注解
		if istioInject, ok := cr.Annotations[IstioInjectAnnotationKey]; ok {
			if istioInject == "true" {
				enable = true
			}
		} else {
			// 没有注解，取configmap默认值
			if istioInject, ok := configMapData["istioInject"]; ok {
				if istioInject == "true" {
					enable = true
				}
			}
		}
	}
	return enable
}

// 如果配置了host使用配置的host，没有配置使用configmap中默认配置，如果configmap没有配置，使用默认值"*.beta.iwosai.com,*.iwosai.com"
func getIngressHosts(configMapData map[string]string, cr *qav1alpha1.SQBApplication) []string {
	var hosts []string
	if len(cr.Spec.Hosts) == 0 {
		domainPostfix, ok := configMapData["domainPostfix"]
		if !ok {
			domainPostfix = "*.beta.iwosai.com,*.iwosai.com"
		}
		hosts = strings.Split(strings.ReplaceAll(domainPostfix, "*", cr.Name), ",")
	} else {
		hosts = cr.Spec.Hosts
	}
	return hosts
}

// 是否开启ingress
func isIngressOpen(configMapData map[string]string, cr *qav1alpha1.SQBApplication) bool {
	enable := false
	if ingressOpen, ok := cr.Annotations[IngressOpenAnnotationKey]; ok {
		if ingressOpen == "true" {
			enable = true
		}
	} else {
		if ingressOpen, ok := configMapData["ingressOpen"]; ok {
			if ingressOpen == "true" {
				enable = true
			}
		}
	}
	return enable
}
