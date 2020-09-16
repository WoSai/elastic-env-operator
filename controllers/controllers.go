package controllers

import (
	"context"
	"crypto/md5"
	"fmt"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"os"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"strings"
)

var (
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
	// 只处理创建和删除
	CreateDeletePredicate = predicate.Funcs{
		UpdateFunc: func(event event.UpdateEvent) bool {
			return false
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return false
		},
	}
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
	// 处理创建删除和annotation的更新
	CreateDeleteAnnotationPredicate = predicate.Funcs{
		UpdateFunc: func(event event.UpdateEvent) bool {
			if event.MetaOld == nil || event.MetaNew == nil || event.ObjectOld == nil || event.ObjectNew == nil {
				return false
			}
			//annotation不变，不处理
			if reflect.DeepEqual(event.MetaOld.GetAnnotations(), event.MetaNew.GetAnnotations()) {
				return false
			}
			return true
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return false
		},
	}
)

// return label
// deprecated
func AddLabel(originLabels map[string]string, key string, value string) map[string]string {
	if len(originLabels) == 0 {
		originLabels = map[string]string{}
	}
	originLabels[key] = value
	return originLabels
}

// return label
func AddLabels(originLabels map[string]string, labels map[string]string) map[string]string {
	if len(originLabels) == 0 {
		originLabels = make(map[string]string)
	}
	for k,v := range labels {
		originLabels[k] = v
	}
	return originLabels
}

//
func GetConfigMapData(client client.Client, ctx context.Context, key client.ObjectKey) map[string]string {
	configmap := &v1.ConfigMap{}
	err := client.Get(ctx, key, configmap)
	if err != nil {
		return map[string]string{}
	}
	return configmap.Data
}

func GetDefaultConfigMapData(client client.Client, ctx context.Context) map[string]string {
	namespace := os.Getenv("CONFIGMAP_NAMESPACE")
	if namespace == "" {
		namespace = "default"
	}
	name := "operator-configmap"
	return GetConfigMapData(client, ctx, types.NamespacedName{Namespace: namespace, Name: name})
}

//
func ContainString(list []string, item string) bool {
	for _, i := range list {
		if i == item {
			return true
		}
	}
	return false
}

// 忽略没有匹配资源的错误
func IgnoreNoMatchError(err error) error {
	if err != nil && !apierrors.IsNotFound(err) && !strings.HasPrefix(err.Error(), "no matches for kind") {
		return err
	}
	return nil
}

//
func GetDeleteCheckSum(cr v12.Object) string {
	salt := os.Getenv("MD5_SALT")
	if salt == "" {
		salt = "0e80b3a3-ad6b-4bc5-a41e-57ea49266417"
	}
	checksum := md5.Sum([]byte(cr.GetName() + salt))
	return fmt.Sprintf("%x", checksum)
}

type ISQBReconciler interface {
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
func HandleReconcile(r ISQBReconciler, ctx context.Context, obj runtime.Object) (ctrl.Result, error) {
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

	err := r.Operate(ctx, obj)
	if err != nil {
		r.ReconcileFail(ctx, obj, err)
	}

	return ctrl.Result{}, err
}
