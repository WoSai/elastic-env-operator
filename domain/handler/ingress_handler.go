package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

type ingressHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx context.Context
}

func NewIngressHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *ingressHandler {
	return &ingressHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func (h *ingressHandler) Operate() error {
	rules := make([]v1beta1.IngressRule, 0)
	for _, host := range h.sqbapplication.Spec.Hosts {
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