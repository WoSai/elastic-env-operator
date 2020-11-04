package handler

import (
	"context"
	"encoding/json"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strconv"
	"strings"
)

type serviceHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx context.Context
}

func NewServiceHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *serviceHandler {
	return &serviceHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func (h *serviceHandler) Operate() error {
	service := &corev1.Service{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	_ = k8sclient.Get(h.ctx, client.ObjectKey{Namespace: service.Namespace, Name: service.Name}, service)
	ports := make([]corev1.ServicePort, 0)
	for _, port := range h.sqbapplication.Spec.Ports {
		port.Name = strings.ToLower(string(port.Protocol)) + "-" + strconv.Itoa(int(port.Port))
		if port.Protocol != corev1.ProtocolUDP {
			port.Protocol = corev1.ProtocolTCP
		}
		ports = append(ports, port)
	}
	service.Spec.Ports = ports
	service.Spec.Selector = map[string]string{
		entity.AppKey: h.sqbapplication.Name,
	}
	if anno, ok := h.sqbapplication.Annotations[entity.ServiceAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &service.Annotations)
	} else {
		service.Annotations = nil
	}
	service.Labels = h.sqbapplication.Labels
	return CreateOrUpdate(h.ctx, service)
}

