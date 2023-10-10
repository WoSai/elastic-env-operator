package handler

import (
	"context"
	"fmt"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	"k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ingressHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	sqbdeployment  *qav1alpha1.SQBDeployment
	ctx            context.Context
}

func NewSqbapplicationIngressHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *ingressHandler {
	return &ingressHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func NewSqbdeploymentIngressHandler(sqbdeployment *qav1alpha1.SQBDeployment, ctx context.Context) *ingressHandler {
	return &ingressHandler{sqbdeployment: sqbdeployment, ctx: ctx}
}

// 1 服务名+nginx class + host唯一对应一个ingress
// 2 服务相同、class相同、host相同，只是path不同，认为应该配置在同一个ingress
// 3 如果确定不同path需要不同的ingress annotation而要配置在不同的ingress中的，这些情况手动配置
func (h *ingressHandler) CreateOrUpdateForSqbapplication() error {
	pathType := v1.PathTypeImplementationSpecific
	ingressNames := make([]string, len(h.sqbapplication.Spec.Domains))
	for i, domain := range h.sqbapplication.Spec.Domains {
		ingress := &v1.Ingress{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: h.sqbapplication.Namespace,
				Name:      getIngressName(h.sqbapplication.Name, domain.Class, domain.Host),
			},
		}
		ingressNames[i] = ingress.Name
		err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		paths := make([]v1.HTTPIngressPath, 0)
		// 开启istio并且有istio-ingressgateway组件
		if IsIstioInject(h.sqbapplication) && HasIstioIngressGateway() {
			path := v1.HTTPIngressPath{
				Backend: v1.IngressBackend{
					Service: &v1.IngressServiceBackend{
						Name: "istio-ingressgateway" + "-" + h.sqbapplication.Namespace,
						Port: v1.ServiceBackendPort{Number: 80},
					},
				},
				PathType: &pathType,
			}
			paths = append(paths, path)
		} else {
			for _, subpath := range h.sqbapplication.Spec.Subpaths {
				path := v1.HTTPIngressPath{
					Path: subpath.Path,
					Backend: v1.IngressBackend{
						Service: &v1.IngressServiceBackend{
							Name: subpath.ServiceName,
							Port: v1.ServiceBackendPort{Number: int32(subpath.ServicePort)},
						},
					},
					PathType: &pathType,
				}
				paths = append(paths, path)
			}
			// https://stackoverflow.com/questions/49829452/why-ingress-serviceport-can-be-port-and-targetport-of-service
			// 使用target port而不是service port
			var servicePort intstr.IntOrString
			if ports := h.sqbapplication.Spec.Ports; len(ports) == 0 {
				servicePort = intstr.FromInt(80)
			} else {
				servicePort = ports[0].TargetPort
			}
			// 默认路由
			path := v1.HTTPIngressPath{
				Backend: v1.IngressBackend{
					Service: &v1.IngressServiceBackend{
						Name: h.sqbapplication.Name,
						Port: v1.ServiceBackendPort{
							Number: servicePort.IntVal,
						},
					},
				},
				PathType: &pathType,
			}
			paths = append(paths, path)
		}
		rule := v1.IngressRule{
			Host: domain.Host,
			IngressRuleValue: v1.IngressRuleValue{
				HTTP: &v1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		}
		ingress.Spec.Rules = []v1.IngressRule{rule}
		ingress.Labels = util.MergeStringMap(ingress.Labels, map[string]string{
			entity.AppKey:   h.sqbapplication.Name,
			entity.GroupKey: h.sqbapplication.Labels[entity.GroupKey],
		})
		if len(domain.Annotation) != 0 {
			ingress.Annotations = util.MergeStringMap(ingress.Annotations, domain.Annotation)
		}
		ingress.Annotations = util.MergeStringMap(ingress.Annotations, map[string]string{
			entity.IngressClassAnnotationKey: domain.Class,
		})
		if err = CreateOrUpdate(h.ctx, ingress); err != nil {
			return err
		}
	}

	// 如果ingress的host没有包含在domainHosts中，且ingress是自动生成的，则删除该ingress
	ingressList := &v1.IngressList{}
	err := k8sclient.List(h.ctx, ingressList, &client.ListOptions{
		Namespace:     h.sqbapplication.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: h.sqbapplication.Name}),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	for _, ingress := range ingressList.Items {
		if h.isAutoIngress(ingress) && !util.ContainString(ingressNames, ingress.Name) {
			if err = Delete(h.ctx, &ingress); err != nil {
				return err
			}
		}
	}
	return nil
}

// 外网特殊入口创建新的ingress
func (h *ingressHandler) CreateOrUpdateForSqbdeployment() error {
	ingressClass := SpecialVirtualServiceIngress(h.sqbdeployment)
	pathType := v1.PathTypeImplementationSpecific
	host := entity.ConfigMapData.GetDomainNameByClass(h.sqbdeployment.Name, SpecialVirtualServiceIngress(h.sqbdeployment))
	ingress := &v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: h.sqbdeployment.Namespace,
			Name:      getIngressName(h.sqbdeployment.Labels[entity.AppKey], ingressClass, host),
		},
	}
	if err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	ingress.Labels = util.MergeStringMap(ingress.Labels, h.sqbdeployment.Labels)

	rule := v1.IngressRule{
		Host: host,
		IngressRuleValue: v1.IngressRuleValue{
			HTTP: &v1.HTTPIngressRuleValue{
				Paths: []v1.HTTPIngressPath{
					{
						Backend: v1.IngressBackend{
							Service: &v1.IngressServiceBackend{
								Name: "istio-ingressgateway" + "-" + h.sqbdeployment.Namespace,
								Port: v1.ServiceBackendPort{Number: 80},
							},
						},
						PathType: &pathType,
					},
				},
			},
		},
	}
	ingress.Spec.Rules = []v1.IngressRule{rule}
	return CreateOrUpdate(h.ctx, ingress)
}

func (h *ingressHandler) DeleteForSqbapplication() error {
	ingressList := &v1.IngressList{}
	err := k8sclient.List(h.ctx, ingressList, &client.ListOptions{
		Namespace:     h.sqbapplication.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: h.sqbapplication.Name}),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	for _, ingress := range ingressList.Items {
		if !h.isAutoIngress(ingress) {
			continue
		}
		if err = Delete(h.ctx, &ingress); err != nil {
			return err
		}
	}
	return nil
}

func (h *ingressHandler) DeleteForSqbdeployment() error {
	ingressClass := SpecialVirtualServiceIngress(h.sqbdeployment)
	host := entity.ConfigMapData.GetDomainNameByClass(h.sqbdeployment.Name, ingressClass)
	ingress := &v1.Ingress{ObjectMeta: metav1.ObjectMeta{
		Namespace: h.sqbdeployment.Namespace,
		Name:      getIngressName(h.sqbdeployment.Labels[entity.AppKey], ingressClass, host),
	}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress)
	if err != nil {
		return client.IgnoreNotFound(err)
	}
	if err = Delete(h.ctx, ingress); err != nil {
		return err
	}
	return nil
}

func (h *ingressHandler) Handle() error {
	if h.sqbapplication != nil {
		if deleted, _ := IsDeleted(h.sqbapplication); deleted || len(h.sqbapplication.Spec.Domains) == 0 {
			return h.DeleteForSqbapplication()
		}
		if !IsIngressOpen(h.sqbapplication) {
			return h.DeleteForSqbapplication()
		}
		return h.CreateOrUpdateForSqbapplication()
	}
	if h.sqbdeployment != nil {
		if deleted, _ := IsDeleted(h.sqbdeployment); deleted {
			return h.DeleteForSqbdeployment()
		}
		if h.sqbdeployment.Annotations[entity.PublicEntryAnnotationKey] != "true" {
			return h.DeleteForSqbdeployment()
		}
		return h.CreateOrUpdateForSqbdeployment()
	}
	return nil
}

// getIngressName, 生成ingress的名称
func getIngressName(appName, nginxClass, host string) string {
	return fmt.Sprintf("%s.%s.%s", appName, nginxClass, host)
}

// isAutoIngressName 判断一个ingress是否是自动生成的
func (h *ingressHandler) isAutoIngress(ingress v1.Ingress) bool {
	if ingress.Annotations == nil || ingress.Annotations[entity.IngressClassAnnotationKey] == "" || len(ingress.Spec.Rules) < 1 {
		return false
	}
	// 新规则
	if getIngressName(h.sqbapplication.Name, ingress.Annotations[entity.IngressClassAnnotationKey],
		ingress.Spec.Rules[0].Host) == ingress.Name {
		return true
	}
	// 老规则
	if fmt.Sprintf("%s-%s", h.sqbapplication.Name, ingress.Annotations[entity.IngressClassAnnotationKey]) == ingress.Name {
		return true
	}
	return false
}
