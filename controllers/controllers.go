package controllers

import (
	"context"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"k8s.io/apimachinery/pkg/runtime"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	XEnvFlag                     = "x-env-flag"
	AppKey                       = "app"
	PlaneKey                     = "plane"
	TeamKey                      = "team"
	SqbplaneFinalizer            = "SQBPLANE"
	SqbdeploymentFinalizer       = "SQBDEPLOYMENT"
	SqbapplicationFinalizer      = "SQBAPPLICATION"
	ExplicitDeleteAnnotationKey  = "qa.shouqianba.com/delete"
	IstioInjectAnnotationKey     = "qa.shouqianba.com/istio-inject"
	IngressOpenAnnotationKey     = "qa.shouqianba.com/ingress-open"
	PublicEntryAnnotationKey     = "qa.shouqianba.com/public-entry"
	DeploymentAnnotationKey      = "qa.shouqianba.com/passthrough-deployment"
	PodAnnotationKey             = "qa.shouqianba.com/passthrough-pod"
	ServiceAnnotationKey         = "qa.shouqianba.com/passthrough-service"
	IngressAnnotationKey         = "qa.shouqianba.com/passthrough-ingress"
	DestinationRuleAnnotationKey = "qa.shouqianba.com/passthrough-destinationrule"
	VirtualServiceAnnotationKey  = "qa.shouqianba.com/passthrough-virtualservice"
)

var (
	// 处理创建删除和generation、annotation的更新
	GenerationAnnotationPredicate = predicate.Funcs{
		UpdateFunc: func(event event.UpdateEvent) bool {
			if event.MetaOld == nil || event.MetaNew == nil || event.ObjectOld == nil || event.ObjectNew == nil {
				return false
			}
			//generation不变且annotation也不变，不处理
			if event.MetaNew.GetGeneration() == event.MetaOld.GetGeneration() &&
				reflect.DeepEqual(event.MetaOld.GetAnnotations(), event.MetaNew.GetAnnotations()) {
				return false
			}
			return true
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return false
		},
	}
)

type ISQBReconciler interface {
	//
	GetInstance(ctx context.Context, req ctrl.Request) (runtime.Object, error)
	// 初始化逻辑
	IsInitialized(ctx context.Context, obj runtime.Object) (bool, error)
	// 正常处理逻辑
	Operate(ctx context.Context, obj runtime.Object) error
	// 处理失败后逻辑
	ReconcileFail(ctx context.Context, obj runtime.Object, err error)
	// 删除逻辑
	IsDeleting(ctx context.Context, obj runtime.Object) (bool, error)
}

// reconcile公共逻辑流程
func HandleReconcile(r ISQBReconciler, ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if !entity.ConfigMapData.Initialized {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	obj, err := r.GetInstance(ctx, req)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if yes, err := r.IsInitialized(ctx, obj); !yes {
		if err != nil {
			r.ReconcileFail(ctx, obj, err)
		}

		return ctrl.Result{Requeue: true}, nil
	}

	if yes, err := r.IsDeleting(ctx, obj); yes {
		if err != nil {
			r.ReconcileFail(ctx, obj, err)
		}
		return ctrl.Result{}, err
	}

	err = r.Operate(ctx, obj)
	if err != nil {
		r.ReconcileFail(ctx, obj, err)
	}

	return ctrl.Result{}, err
}
