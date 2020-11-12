package handler

import (
	"context"
	"encoding/json"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"k8s.io/api/extensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ingressHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx            context.Context
}

func NewIngressHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *ingressHandler {
	return &ingressHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func (h *ingressHandler) CreateOrUpdate() error {
	ingress := &v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: ingress.Namespace, Name: ingress.Name}, ingress)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	rules := make([]v1beta1.IngressRule, 0)
	for _, host := range h.sqbapplication.Spec.Hosts {
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
			path := v1beta1.HTTPIngressPath{
				Backend: v1beta1.IngressBackend{
					ServiceName: h.sqbapplication.Name,
					ServicePort: intstr.FromInt(80),
				},
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
	ingress.Spec.Rules = rules
	if anno, ok := h.sqbapplication.Annotations[entity.IngressAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &ingress.Annotations)
	} else {
		ingress.Annotations = nil
	}
	return CreateOrUpdate(h.ctx, ingress)
}

func (h *ingressHandler) Delete() error {
	ingress := &v1beta1.Ingress{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	return Delete(h.ctx, ingress)
}

func (h *ingressHandler) Handle() error {
	if IsExplicitDelete(h.sqbapplication) {
		return h.Delete()
	}
	if !IsIngressOpen(h.sqbapplication) {
		return h.Delete()
	}
	return h.CreateOrUpdate()
}
