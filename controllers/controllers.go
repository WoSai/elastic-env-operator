package controllers

import (
	"context"
	"crypto/md5"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
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
	configMapData map[string]string
)

//
func MergeStringMap(base map[string]string, toMerge map[string]string) map[string]string {
	if len(base) == 0 {
		base = make(map[string]string)
	}
	for k, v := range toMerge {
		base[k] = v
	}
	return base
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

func getIstioTimeout() int64 {
	timeout, ok := configMapData["istioTimeout"]
	if !ok {
		timeout = "90"
	}
	routeTimeout, err := strconv.Atoi(timeout)
	if err != nil {
		routeTimeout = 90
	}
	return int64(routeTimeout)
}

func getIstioGateways() []string {
	if gateways, ok := configMapData["istioGateways"]; ok {
		return strings.Split(gateways, ",")
	}
	return []string{"mesh"}
}

func getDefaultDomainName(sqbapplicationName string) []string {
	domainPostfix, ok := configMapData["domainPostfix"]
	if !ok {
		domainPostfix = "*.beta.iwosai.com,*.iwosai.com"
	}
	hosts := strings.Split(strings.ReplaceAll(domainPostfix, "*", sqbapplicationName), ",")
	return hosts
}

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
	if len(configMapData) == 0 {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	obj, err := r.GetInstance(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
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

func init() {
	go func() {
		timer := time.NewTimer(60 * time.Second)
		for {
			if len(configMapData) == 0 {
				select {
				case <-timer.C:
					panic("operator configmap is not valid")
				case <-time.After(time.Second):
				}
			} else {
				timer.Stop()
				break
			}
		}
	}()
}
