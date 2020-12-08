package handler

import (
	"context"
	"fmt"
	"github.com/go-logr/logr"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"time"
)

var (
	k8sclient client.Client
	log       logr.Logger
	k8sScheme *runtime.Scheme
)

type (
	runtimeObj interface {
		runtime.Object
		metav1.Object
	}

	SQBReconciler interface {
		GetInstance() (runtimeObj, error)
		// IsInitialized 第一次创建后需要执行的操作
		IsInitialized(runtimeObj) (bool, error)
		Operate(runtimeObj) error
		ReconcileFail(runtimeObj, error)
	}

	SQBHandler interface {
		Handle() error
	}
)

func SetK8sClient(c client.Client) {
	k8sclient = c
}

func SetK8sLog(l logr.Logger) {
	log = l
}

func SetK8sScheme(s *runtime.Scheme) {
	k8sScheme = s
}

func HandleReconcile(r SQBReconciler) (ctrl.Result, error) {
	if !entity.ConfigMapData.Initialized {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}
	obj, err := r.GetInstance()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			r.ReconcileFail(obj, err)
			return ctrl.Result{}, util.IgnoreInvalidError(err)
		}
	}

	generation := obj.GetGeneration()
	if yes, err := r.IsInitialized(obj); !yes {
		if err != nil {
			r.ReconcileFail(obj, err)
			return ctrl.Result{}, util.IgnoreInvalidError(err)
		}
		if generation != obj.GetGeneration() {
			return ctrl.Result{}, nil
		}
	}

	if err = r.Operate(obj); err != nil {
		r.ReconcileFail(obj, err)
		return ctrl.Result{}, util.IgnoreInvalidError(err)
	}
	return ctrl.Result{}, nil
}

func CreateOrUpdate(ctx context.Context, obj runtimeObj) error {
	kind, _ := apiutil.GVKForObject(obj, k8sScheme)
	if obj.GetCreationTimestamp().Time.IsZero() {
		err := k8sclient.Create(ctx, obj)
		log.Info("create obj", "kind", kind,
			"namespace", obj.GetNamespace(), "name", obj.GetName(), "error", err)
		return err
	}
	err := k8sclient.Update(ctx, obj)
	log.Info("update obj", "kind", kind,
		"namespace", obj.GetNamespace(), "name", obj.GetName(), "error", err)
	return err
}

func UpdateStatus(ctx context.Context, obj runtimeObj) error {
	kind, _ := apiutil.GVKForObject(obj, k8sScheme)
	err := k8sclient.Status().Update(ctx, obj)
	log.Info("update obj status", "kind", kind,
		"namespace", obj.GetNamespace(), "name", obj.GetName(), "error", err)
	return err
}

func Delete(ctx context.Context, obj runtimeObj) error {
	kind, _ := apiutil.GVKForObject(obj, k8sScheme)
	err := k8sclient.Delete(ctx, obj)
	log.Info("delete obj", "kind", kind,
		"namespace", obj.GetNamespace(), "name", obj.GetName(), "error", err)
	return client.IgnoreNotFound(err)
}

func IsDeleted(obj runtimeObj) (bool, error) {
	if deleteCheckSum, ok := obj.GetAnnotations()[entity.ExplicitDeleteAnnotationKey]; ok {
		if deleteCheckSum == util.GetDeleteCheckSum(obj.GetName()) {
			return true, nil
		} else {
			return false, fmt.Errorf("delete annotation %s is wrong", deleteCheckSum)
		}
	}
	return false, nil
}
