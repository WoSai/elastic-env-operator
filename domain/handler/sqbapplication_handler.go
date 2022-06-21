package handler

import (
	"context"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	ctrl "sigs.k8s.io/controller-runtime"
)

type sqbApplicationHandler struct {
	req ctrl.Request
	ctx context.Context
}

func NewSqbApplicationHanlder(req ctrl.Request, ctx context.Context) *sqbApplicationHandler {
	return &sqbApplicationHandler{req: req, ctx: ctx}
}

func (h *sqbApplicationHandler) GetInstance() (runtimeObj, error) {
	in := &qav1alpha1.SQBApplication{}
	err := k8sclient.Get(h.ctx, h.req.NamespacedName, in)
	return in, err
}

func (h *sqbApplicationHandler) IsInitialized(_ runtimeObj) (bool, error) {
	return true, nil
}

func (h *sqbApplicationHandler) Operate(obj runtimeObj) error {
	in := obj.(*qav1alpha1.SQBApplication)
	deleted, err := IsDeleted(in)
	if err != nil {
		return err
	}
	// 补充默认值
	for i, domain := range in.Spec.Domains {
		if domain.Host == "" {
			domain.Host = entity.ConfigMapData.GetDomainNameByClass(in.Name, domain.Class)
			in.Spec.Domains[i] = domain
		}
	}

	handlers := []SQBHandler{
		NewServiceHandler(in, h.ctx),
		NewSqbapplicationIngressHandler(in, h.ctx),
		NewDestinationRuleHandler(in, h.ctx),
		//NewVirtualServiceHandler(in, h.ctx),
		//NewServiceMonitorHandler(in, h.ctx),
		NewSqbDeploymentListHandlerForSqbapplication(in, h.ctx),
		NewVMServiceScrapeHandler(in, h.ctx),
	}

	for _, handler := range handlers {
		if err := handler.Handle(); err != nil {
			return err
		}
	}

	if deleted {
		return Delete(h.ctx, in)
	} else if in.Status.ErrorInfo != "" {
		in.Status.ErrorInfo = ""
		return UpdateStatus(h.ctx, in)
	}
	return nil
}

func (h *sqbApplicationHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*qav1alpha1.SQBApplication)
	in.Status.ErrorInfo = err.Error()
	_ = UpdateStatus(h.ctx, in)
}

// 判断应用是否启用istio逻辑：
// 1.如果集群装了istio且有注解，根据注解
// 2.如果集群装了istio但没有注解，根据集群默认配置
// 3.如果集群没有装istio，不启用istio
func IsIstioInject(sqbapplication *qav1alpha1.SQBApplication) bool {
	if entity.ConfigMapData.IstioEnable() {
		if istioInject, ok := sqbapplication.Annotations[entity.IstioInjectAnnotationKey]; ok {
			return istioInject == "true"
		}
		return entity.ConfigMapData.IstioInject()
	}
	return false
}

// 判断应用是否启用ingress逻辑：
// 1.有注解，根据注解
// 2.没有注解，根据默认配置
func IsIngressOpen(sqbapplication *qav1alpha1.SQBApplication) bool {
	if is, ok := sqbapplication.Annotations[entity.IngressOpenAnnotationKey]; ok {
		return is == "true"
	}
	return entity.ConfigMapData.IngressOpen()
}
