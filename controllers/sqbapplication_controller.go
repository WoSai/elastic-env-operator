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
	jsonpatch "github.com/evanphx/json-patch"
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
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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

var _ ISQBReconciler = &SQBApplicationReconciler{}

// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbapplications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=qa.shouqianba.com,resources=sqbapplications/status,verbs=get;update;patch

func (r *SQBApplicationReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	return HandleReconcile(r, ctx, req)
}

func (r *SQBApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&qav1alpha1.SQBApplication{}, builder.WithPredicates(GenerationAnnotationPredicate)).
		Watches(&source.Kind{Type: &v12.Deployment{}},
			&handler.EnqueueRequestsFromMapFunc{ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
				return []reconcile.Request{
					{NamespacedName: types.NamespacedName{
						Name:      a.Meta.GetLabels()[AppKey],
						Namespace: a.Meta.GetNamespace(),
					}},
				}
			})},
			builder.WithPredicates(CreateDeleteAnnotationPredicate)).
		Complete(r)
}

func (r *SQBApplicationReconciler) GetInstance(ctx context.Context, req ctrl.Request) (runtime.Object, error) {
	instance := &qav1alpha1.SQBApplication{}
	err := r.Get(ctx, req.NamespacedName, instance)
	return instance, client.IgnoreNotFound(err)
}

func (r *SQBApplicationReconciler) IsInitialized(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBApplication)
	if cr.Status.Initialized == true {
		return true, nil
	}
	if len(cr.Annotations) == 0 {
		cr.Annotations = make(map[string]string)
	}
	if globalDefaultDeploy, ok := configMapData["globalDefaultDeploy"]; ok {
		applicationDeploy, _ := json.Marshal(cr.Spec.DeploySpec)
		applicationDeploy, _ = jsonpatch.MergePatch([]byte(globalDefaultDeploy), applicationDeploy)
		deploy := qav1alpha1.DeploySpec{}
		err := json.Unmarshal(applicationDeploy, &deploy)
		if err != nil {
			return false, err
		}
		cr.Spec.DeploySpec = deploy
	}
	controllerutil.AddFinalizer(cr, SqbapplicationFinalizer)
	cr.Spec.Hosts = getIngressHosts(cr)
	// 添加一条默认的subpath /在最后
	cr.Spec.Subpaths = append(cr.Spec.Subpaths, qav1alpha1.Subpath{
		Path: "/", ServiceName: cr.Name, ServicePort: 80})
	cr.Annotations[IstioInjectAnnotationKey] = r.getIstioInjectionResult(ctx, cr)
	cr.Annotations[IngressAnnotationKey] = r.getIngressOpenResult(cr)
	err := r.Update(ctx, cr)
	if err != nil {
		return false, err
	}
	cr.Status.Initialized = true
	return false, r.Status().Update(ctx, cr)
}

func (r *SQBApplicationReconciler) IsDeleting(ctx context.Context, obj runtime.Object) (bool, error) {
	cr := obj.(*qav1alpha1.SQBApplication)
	if cr.DeletionTimestamp.IsZero() || !controllerutil.ContainsFinalizer(cr, SqbapplicationFinalizer) {
		return false, nil
	}

	var err error

	if deleteCheckSum, ok := cr.Annotations[ExplicitDeleteAnnotationKey]; ok && deleteCheckSum == GetDeleteCheckSum(cr) {
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
		if cr.Annotations[IstioInjectAnnotationKey] == "true" {
			// 如果有istio,删除virtualservice,destinationrule
			destinationrule := &v1beta13.DestinationRule{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
			err = r.Delete(ctx, destinationrule)
			if IgnoreNoMatchError(err) != nil {
				return true, err
			}
			virtualservice := &v1beta13.VirtualService{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
			err = r.Delete(ctx, virtualservice)
			if IgnoreNoMatchError(err) != nil {
				return true, err
			}
		}
		// 删除SQBDeployment和Deployment
		err = DeleteSqbdeploymentByLabel(r.Client, ctx, cr.Namespace, map[string]string{AppKey: cr.Name})
		if err != nil {
			return true, err
		}
		// deployment会触发事件，所以最后删除
		err = DeleteDeploymentByLabel(r.Client, ctx, cr.Namespace, map[string]string{AppKey: cr.Name})
		if err != nil {
			return true, err
		}
	}
	return true, r.RemoveFinalizer(ctx, cr)
}

func (r *SQBApplicationReconciler) Operate(ctx context.Context, obj runtime.Object) error {
	cr := obj.(*qav1alpha1.SQBApplication)
	var err error
	// 判断是否有对应deployment
	deploymentList := &v12.DeploymentList{}
	err = r.List(ctx, deploymentList, &client.ListOptions{Namespace: cr.Namespace, LabelSelector: labels.SelectorFromSet(map[string]string{AppKey: cr.Name})})
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	mirrors := make(map[string]int)
	planes := make(map[string]int)
	for _, deployment := range deploymentList.Items {
		mirrors[deployment.Name] = 1
		if plane, ok := deployment.Labels[PlaneKey]; ok {
			planes[plane] = 1
		}
	}
	cr.Status.Mirrors = mirrors
	cr.Status.Planes = planes
	if len(planes) == 0 {
		if cr.Spec.Image != "" {
			// 如果服务没有部署，创建一个base环境的部署
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
			}
			_, err = controllerutil.CreateOrUpdate(ctx, r.Client, sqbDeployment, func() error {
				sqbDeployment.Spec = qav1alpha1.SQBDeploymentSpec{
					Selector: qav1alpha1.Selector{
						App:   cr.Name,
						Plane: "base",
					},
					DeploySpec: cr.Spec.DeploySpec,
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		cr.Status.ErrorInfo = ""
		return r.Status().Update(ctx, cr)
	}
	//　处理service
	err = r.createOrUpdateService(ctx, cr)
	if err != nil {
		return err
	}
	// 处理istio相关配置
	if cr.Annotations[IstioInjectAnnotationKey] == "true" {
		err := r.handleIstio(ctx, cr, deploymentList)
		if err != nil {
			return err
		}
	} else {
		err := r.handleNoIstio(ctx, cr)
		if err != nil {
			return err
		}
	}
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

func (r *SQBApplicationReconciler) createOrUpdateService(ctx context.Context, cr *qav1alpha1.SQBApplication) error {
	service := &v13.Service{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	_, err := controllerutil.CreateOrUpdate(ctx, r.Client, service, func() error {
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
		if anno, ok := cr.Annotations[ServiceAnnotationKey]; ok {
			err := json.Unmarshal([]byte(anno), &service.Annotations)
			if err != nil {
				return err
			}
		}
		service.Labels = cr.Labels
		return nil
	})
	return err
}

// 处理启用istio的逻辑
func (r *SQBApplicationReconciler) handleIstio(ctx context.Context, cr *qav1alpha1.SQBApplication,
	deploymentList *v12.DeploymentList) error {
	// Ingress
	ingress := &v1beta12.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	if cr.Annotations[IngressOpenAnnotationKey] == "true" {
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
			for _, host := range cr.Spec.Hosts {
				rule := v1beta12.IngressRule{
					Host:             host,
					IngressRuleValue: ingressRule,
				}
				rules = append(rules, rule)
			}
			for _, deployment := range deploymentList.Items {
				if _, ok := deployment.Annotations[PublicEntryAnnotationKey]; ok {
					for _, publicEntry := range getSpecialVirtualServiceHost(&deployment) {
						rule := v1beta12.IngressRule{
							Host:             publicEntry,
							IngressRuleValue: ingressRule,
						}
						rules = append(rules, rule)
					}
				}
			}
			ingress.Spec = v1beta12.IngressSpec{Rules: rules}
			if anno, ok := cr.Annotations[IngressAnnotationKey]; ok {
				err := json.Unmarshal([]byte(anno), &ingress.Annotations)
				if err != nil {
					return err
				}
			}
			ingress.Labels = cr.Labels
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
		virtualserviceHosts := append(cr.Spec.Hosts, cr.Name)
		gateways := getIstioGateways()
		virtualservice.Spec.Hosts = virtualserviceHosts
		virtualservice.Spec.Gateways = gateways
		virtualservice.Spec.Http = getOrGenerateHttpRoutes(virtualservice.Spec.Http, cr.Spec.Subpaths, planes)
		// 处理tcp route
		for _, port := range cr.Spec.Ports {
			if ContainString([]string{"tcp", "mongo", "mysql", "redis"}, strings.ToLower(string(port.Protocol))) {
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
func (r *SQBApplicationReconciler) handleNoIstio(ctx context.Context, cr *qav1alpha1.SQBApplication) error {
	// Ingress
	ingress := &v1beta12.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	if cr.Annotations[IngressOpenAnnotationKey] == "true" {
		_, err := controllerutil.CreateOrUpdate(ctx, r.Client, ingress, func() error {
			rules := make([]v1beta12.IngressRule, 0)
			for _, host := range cr.Spec.Hosts {
				paths := make([]v1beta12.HTTPIngressPath, 0)
				for _, subpath := range cr.Spec.Subpaths {
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
	if IgnoreNoMatchError(err) != nil {
		return err
	}

	destinationrule := &v1beta13.DestinationRule{ObjectMeta: v1.ObjectMeta{Namespace: cr.Namespace, Name: cr.Name}}
	err = r.Delete(ctx, destinationrule)
	if IgnoreNoMatchError(err) != nil {
		return err
	}
	return nil
}

func (r *SQBApplicationReconciler) getIstioInjectionResult(ctx context.Context, cr *qav1alpha1.SQBApplication) string {
	enable := "false"
	istio := &v14.CustomResourceDefinition{}
	err := r.Get(ctx, types.NamespacedName{Namespace: "", Name: "virtualservices.networking.istio.io"}, istio)
	// err==nil 表示集群安装了istio
	if err == nil {
		// 判断application注解
		if istioInject, ok := cr.Annotations[IstioInjectAnnotationKey]; ok {
			enable = istioInject
		} else {
			// 没有注解，取configmap默认值
			if istioInject, ok := configMapData["istioInject"]; ok {
				enable = istioInject
			}
		}
	}
	return enable
}

func (r *SQBApplicationReconciler) getIngressOpenResult(cr *qav1alpha1.SQBApplication) string {
	enable := "false"
	if ingressOpen, ok := cr.Annotations[IngressOpenAnnotationKey]; ok {
		enable = ingressOpen
	} else {
		if ingressOpen, ok := configMapData["ingressOpen"]; ok {
			enable = ingressOpen
		}
	}
	return enable
}

// 根据plane生成DestinationRule的subsets
func generateSubsets(cr *qav1alpha1.SQBApplication, planes map[string]int) []*v1beta14.Subset {
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
func getOrGenerateHttpRoutes(httpRoutes []*v1beta14.HTTPRoute, subpaths []qav1alpha1.Subpath,
	planes map[string]int) []*v1beta14.HTTPRoute {
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
				Timeout: &types2.Duration{Seconds: getIstioTimeout()},
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
		planes["base"] = 1
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
				Timeout: &types2.Duration{Seconds: getIstioTimeout()},
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
	planes map[string]int) []*v1beta14.TCPRoute {
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
		planes["base"] = 1
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

func getIngressHosts(cr *qav1alpha1.SQBApplication) []string {
	hosts := getDefaultDomainName(cr.Name)
	for _, host := range cr.Spec.Hosts {
		if !ContainString(hosts, host) {
			hosts = append(hosts, host)
		}
	}
	return hosts
}
