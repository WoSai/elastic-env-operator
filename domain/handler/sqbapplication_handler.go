package handler

import (
	"context"
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	appv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
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
	// hosts为用户定义+configmap默认配置
	hosts := entity.ConfigMapData.GetDomainNames(in.Name)
	for _, host := range in.Spec.Hosts {
		if !util.ContainString(hosts, host) {
			hosts = append(hosts, host)
		}
	}
	in.Spec.Hosts = hosts
	// 添加一条默认的subpath /在最后
	in.Spec.Subpaths = append(in.Spec.Subpaths, qav1alpha1.Subpath{
		Path: "/", ServiceName: in.Name, ServicePort: 80})
	if len(in.Annotations) == 0 {
		in.Annotations = make(map[string]string)
	}
	in.Annotations[entity.InitializeAnnotationKey] = "true"
	return false, CreateOrUpdate(h.ctx, in)
}

func (h *sqbApplicationHandler) Operate(obj runtimeObj) error {
	in := obj.(*qav1alpha1.SQBApplication)
	// 如果没有被删除，先更新planes和mirrors
	if in.DeletionTimestamp.IsZero() {
		deployments := &appv1.DeploymentList{}
		if err := k8sclient.List(h.ctx, deployments,
			&client.ListOptions{
				Namespace:     in.Namespace,
				LabelSelector: labels.SelectorFromSet(map[string]string{entity.AppKey: in.Name}),
			},
		); err != nil && !apierrors.IsNotFound(err) {
			return err
		}
		// SQBDeployment可能会被删除，所以planes和mirrors以deployment为准
		planes := make(map[string]int)
		mirrors := make(map[string]int)
		for _, deployment := range deployments.Items {
			mirrors[deployment.Name] = 1
			if plane, ok := deployment.Labels[entity.PlaneKey]; ok {
				planes[plane] = 1
			}
		}
		in.Status.Planes = planes
		in.Status.Mirrors = mirrors
		if len(planes) == 0 {
			return h.CreateBase(in)
		}
	}

	handlers := []SQBHanlder{
		NewServiceHandler(in, h.ctx),
		NewIngressHandler(in, h.ctx),
		NewDestinationRuleHandler(in, h.ctx),
		NewVirtualServiceHandler(in, h.ctx),
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
	} else {
		in.Status.ErrorInfo = ""
		return k8sclient.Status().Update(h.ctx, in)
	}
}

func (h *sqbApplicationHandler) ReconcileFail(obj runtimeObj, err error) {
	in := obj.(*qav1alpha1.SQBApplication)
	in.Status.ErrorInfo = err.Error()
	_ = k8sclient.Status().Update(h.ctx, in)
}

func (h *sqbApplicationHandler) CreateBase(sqbapplication *qav1alpha1.SQBApplication) error {
	if sqbapplication.Spec.Image != "" {
		// 创建对应的base环境服务
		sqbplane := &qav1alpha1.SQBPlane{
			ObjectMeta: metav1.ObjectMeta{Namespace: sqbapplication.Namespace, Name: "base"},
			Spec: qav1alpha1.SQBPlaneSpec{
				Description: "base",
			},
		}
		sqbdeployment := &qav1alpha1.SQBDeployment{
			ObjectMeta: metav1.ObjectMeta{Namespace: sqbapplication.Namespace, Name: util.GetSubsetName(sqbapplication.Name, sqbplane.Name)},
			Spec: qav1alpha1.SQBDeploymentSpec{
				Selector: qav1alpha1.Selector{
					App:   sqbapplication.Name,
					Plane: sqbplane.Name,
				},
			},
		}
		if err := CreateOrUpdate(h.ctx, sqbplane); err != nil {
			return err
		}
		if err := CreateOrUpdate(h.ctx, sqbdeployment); err != nil {
			return err
		}
	} else if sqbapplication.Status.ErrorInfo != "" {
		sqbapplication.Status.ErrorInfo = ""
		return k8sclient.Status().Update(h.ctx, sqbapplication)
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
