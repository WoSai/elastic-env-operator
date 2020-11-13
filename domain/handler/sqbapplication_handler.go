package handler

import (
	"context"
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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

func (h *sqbApplicationHandler) IsInitialized(obj runtimeObj) (bool, error) {
	in := obj.(*qav1alpha1.SQBApplication)
	if in.Annotations[entity.InitializeAnnotationKey] == "true" {
		return true, nil
	}

	if globalDefaultDeploy, ok := entity.ConfigMapData.GlobalDeploy(); ok {
		applicationDeploy, _ := json.Marshal(in.Spec.DeploySpec)
		applicationDeploy, _ = jsonpatch.MergePatch([]byte(globalDefaultDeploy), applicationDeploy)
		deploy := qav1alpha1.DeploySpec{}
		if err := json.Unmarshal(applicationDeploy, &deploy); err == nil {
			in.Spec.DeploySpec = deploy
		}
	}
	controllerutil.AddFinalizer(in, entity.Finalizer)
	if len(in.Spec.Domains) == 0 {
		for k,v := range entity.ConfigMapData.GetDomainNames(in.Name) {
			in.Spec.Domains = append(in.Spec.Domains, qav1alpha1.Domain{Class: k, Host: v})
		}
	}
	if len(in.Annotations) == 0 {
		in.Annotations = make(map[string]string)
	}
	in.Annotations[entity.InitializeAnnotationKey] = "true"
	return false, CreateOrUpdate(h.ctx, in)
}

func (h *sqbApplicationHandler) Operate(obj runtimeObj) error {
	in := obj.(*qav1alpha1.SQBApplication)
	if in.DeletionTimestamp.IsZero() && len(in.Status.Planes) == 0 {
		return h.CreateBase(in)
	}

	handlers := []SQBHandler{
		NewServiceHandler(in, h.ctx),
		NewSqbapplicationIngressHandler(in, h.ctx),
		NewDestinationRuleHandler(in, h.ctx),
		NewVirtualServiceHandler(in, h.ctx),
		NewServiceMonitorHandler(in, h.ctx),
		NewSqbDeploymentListHandler(in, nil, h.ctx),
		NewDeploymentListHandler(in, nil, h.ctx),
	}

	for _, handler := range handlers {
		if err := handler.Handle(); err != nil {
			return err
		}
	}

	if !in.DeletionTimestamp.IsZero() {
		controllerutil.RemoveFinalizer(in, entity.Finalizer)
		return CreateOrUpdate(h.ctx, in)
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

func (h *sqbApplicationHandler) CreateBase(sqbapplication *qav1alpha1.SQBApplication) error {
	if sqbapplication.Spec.Image != "" {
		// 创建对应的base环境服务
		sqbplane := &qav1alpha1.SQBPlane{
			ObjectMeta: metav1.ObjectMeta{Namespace: sqbapplication.Namespace, Name: "base"},
		}
		err := k8sclient.Get(h.ctx, client.ObjectKey{Namespace: sqbplane.Namespace, Name: sqbplane.Name}, sqbplane)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		sqbplane.Spec = qav1alpha1.SQBPlaneSpec{
			Description: "base",
		}
		if err = CreateOrUpdate(h.ctx, sqbplane); err != nil {
			return err
		}

		sqbdeployment := &qav1alpha1.SQBDeployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: sqbapplication.Namespace, Name: util.GetSubsetName(sqbapplication.Name, sqbplane.Name)},
		}
		err = k8sclient.Get(h.ctx, client.ObjectKey{Namespace: sqbdeployment.Namespace, Name: sqbdeployment.Name}, sqbdeployment)
		if err != nil && !apierrors.IsNotFound(err) {
			return err
		}

		sqbdeployment.Spec = qav1alpha1.SQBDeploymentSpec{
			Selector: qav1alpha1.Selector{
				App:   sqbapplication.Name,
				Plane: sqbplane.Name,
			},
		}
		if err = CreateOrUpdate(h.ctx, sqbdeployment); err != nil {
			return err
		}
	} else if sqbapplication.Status.ErrorInfo != "" {
		sqbapplication.Status.ErrorInfo = ""
		return UpdateStatus(h.ctx, sqbapplication)
	}
	return nil
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

// 判断是否启用ServiceMonitor逻辑
// 1. 安装了prometheus-operator且有注解，按注解，其他情况不启用
// 2. 安装了prometheus-operator但没有注解，或者没有安装operator，不启用
func IsServiceMonitorOpen(sqbapplication *qav1alpha1.SQBApplication) bool {
	if entity.ConfigMapData.IsServiceMonitorEnable() {
		if serviceMonitor, ok := sqbapplication.Annotations[entity.ServiceMonitorAnnotationKey]; ok {
			return serviceMonitor == "true"
		}
	}
	return false
}