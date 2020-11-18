package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	"k8s.io/api/extensions/v1beta1"
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

func (h *ingressHandler) CreateOrUpdateForSqbapplication() error {
	domainHosts := make([]string, 0)
	for _, domain := range h.sqbapplication.Spec.Domains {
		ingress := &v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name + "-" + domain.Class}}
		err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		paths := make([]v1beta1.HTTPIngressPath, 0)
		if IsIstioInject(h.sqbapplication) {
			path := v1beta1.HTTPIngressPath{
				Backend: v1beta1.IngressBackend{
					ServiceName: "istio-ingressgateway" + "-" + h.sqbapplication.Namespace,
					ServicePort: intstr.FromInt(80),
				},
			}
			paths = append(paths, path)
		} else {
			for _, subpath := range h.sqbapplication.Spec.Subpaths {
				path := v1beta1.HTTPIngressPath{
					Path: subpath.Path,
					Backend: v1beta1.IngressBackend{
						ServiceName: subpath.ServiceName,
						ServicePort: intstr.FromInt(subpath.ServicePort),
					},
				}
				paths = append(paths, path)
			}
			// 默认路由
			path := v1beta1.HTTPIngressPath{
				Backend: v1beta1.IngressBackend{
					ServiceName: h.sqbapplication.Name,
					ServicePort: intstr.FromInt(80),
				},
			}
			paths = append(paths, path)
		}
		host := domain.Host
		domainHosts = append(domainHosts, host)
		if host == "" {
			host = entity.ConfigMapData.GetDomainNameByClass(h.sqbapplication.Name, domain.Class)
		}
		newrule := v1beta1.IngressRule{
			Host: host,
			IngressRuleValue: v1beta1.IngressRuleValue{
				HTTP: &v1beta1.HTTPIngressRuleValue{
					Paths: paths,
				},
			},
		}
		// 与newrule的host不同的rule保留，是special virtualservice的入口或手动配置
		rules := make([]v1beta1.IngressRule, 0)
		for _, rule := range ingress.Spec.Rules {
			if rule.Host != newrule.Host {
				rules = append(rules, rule)
			}
		}
		rules = append(rules, newrule)
		ingress.Spec.Rules = rules
		ingress.Labels = util.MergeStringMap(ingress.Labels, map[string]string{
			entity.AppKey:   h.sqbapplication.Name,
			entity.GroupKey: h.sqbapplication.Labels[entity.GroupKey],
		})
		if len(domain.Annotation) != 0 {
			ingress.Annotations = domain.Annotation
		} else {
			ingress.Annotations = nil
		}
		if err = CreateOrUpdate(h.ctx, ingress); err != nil {
			return err
		}
	}

	ingressList := &v1beta1.IngressList{}
	err := k8sclient.List(h.ctx, ingressList, &client.ListOptions{
		Namespace:     h.sqbapplication.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: h.sqbapplication.Name}),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}

	loopIngress:
	for _, ingress := range ingressList.Items {
		for _, rule := range ingress.Spec.Rules {
			if util.ContainString(domainHosts, rule.Host) {
				continue loopIngress
			}
		}
		if err = Delete(h.ctx, &ingress); err != nil {
			return err
		}
	}
	return nil
}

func (h *ingressHandler) CreateOrUpdateForSqbdeployment() error {
	ingressClass := SpecialVirtualServiceIngress(h.sqbdeployment)
	ingress := &v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{
		Namespace: h.sqbdeployment.Namespace,
		Name:      h.sqbdeployment.Labels[entity.AppKey] + "-" + ingressClass,
	}}
	if err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress); err != nil {
		return err
	}

	newrule := v1beta1.IngressRule{
		Host: entity.ConfigMapData.GetDomainNameByClass(h.sqbdeployment.Name, SpecialVirtualServiceIngress(h.sqbdeployment)),
		IngressRuleValue: v1beta1.IngressRuleValue{
			HTTP: &v1beta1.HTTPIngressRuleValue{
				Paths: []v1beta1.HTTPIngressPath{
					{
						Backend: v1beta1.IngressBackend{
							ServiceName: "istio-ingressgateway" + "-" + h.sqbdeployment.Namespace,
							ServicePort: intstr.FromInt(80),
						},
					},
				},
			},
		},
	}
	rules := make([]v1beta1.IngressRule, 0)
	for _, rule := range ingress.Spec.Rules {
		if rule.Host != newrule.Host {
			rules = append(rules, rule)
		}
	}
	rules = append(rules, newrule)
	ingress.Spec.Rules = rules
	return CreateOrUpdate(h.ctx, ingress)
}

func (h *ingressHandler) DeleteForSqbapplication() error {
	ingressList := &v1beta1.IngressList{}
	err := k8sclient.List(h.ctx, ingressList, &client.ListOptions{
		Namespace:     h.sqbapplication.Namespace,
		LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: h.sqbapplication.Name}),
	})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	for _, ingress := range ingressList.Items {
		if err = Delete(h.ctx, &ingress); err != nil {
			return err
		}
	}
	return nil
}

func (h *ingressHandler) DeleteForSqbdeployment() error {
	ingressClass := SpecialVirtualServiceIngress(h.sqbdeployment)
	host := entity.ConfigMapData.GetDomainNameByClass(h.sqbdeployment.Name, ingressClass)
	ingress := &v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{
		Namespace: h.sqbdeployment.Namespace,
		Name:      h.sqbdeployment.Labels[entity.AppKey] + "-" + ingressClass,
	}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress)
	if err != nil {
		return client.IgnoreNotFound(err)
	} else {
		rules := make([]v1beta1.IngressRule, 0)
		for _, rule := range ingress.Spec.Rules {
			if rule.Host != host {
				rules = append(rules, rule)
			}
		}
		ingress.Spec.Rules = rules
		return CreateOrUpdate(h.ctx, ingress)
	}
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
