package handler

import (
	"context"
	"encoding/json"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	istioapi "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type destinationRuleHandler struct {
	sqbapplication *qav1alpha1.SQBApplication
	ctx context.Context
}

func NewDestinationRuleHandler(sqbapplication *qav1alpha1.SQBApplication, ctx context.Context) *destinationRuleHandler {
	return &destinationRuleHandler{sqbapplication: sqbapplication, ctx: ctx}
}

func (h *destinationRuleHandler) Operate() error {
	destinationrule := &istio.DestinationRule{ObjectMeta: metav1.ObjectMeta{Namespace: h.sqbapplication.Namespace, Name: h.sqbapplication.Name}}
	err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: destinationrule.Namespace, Name: destinationrule.Name}, destinationrule)
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	destinationrule.Spec.Host = h.sqbapplication.Name

	subsets := make([]*istioapi.Subset, 0)
	for plane := range h.sqbapplication.Status.Planes {
		subsets = append(subsets, &istioapi.Subset{
			Name: util.GetSubsetName(h.sqbapplication.Name, plane),
			Labels: map[string]string{
				entity.PlaneKey: plane,
			},
		})
	}

	destinationrule.Spec.Subsets = subsets
	if anno, ok := h.sqbapplication.Annotations[entity.DestinationRuleAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &destinationrule.Annotations)
	} else {
		destinationrule.Annotations = nil
	}
	return CreateOrUpdate(h.ctx, destinationrule)
}
