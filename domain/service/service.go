package service

import (
	"context"
	"github.com/wosai/elastic-env-operator/domain/entity"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"time"
)

var (
	k8sclient client.Client
)

func SetK8sClient(c client.Client) {
	k8sclient = c
}

type runtimeObj interface {
	runtime.Object
	metav1.Object
}

type ISQBReconciler interface {
	//
	GetInstance() (runtimeObj, error)
	// 初始化逻辑
	IsInitialized(runtimeObj) (bool, error)
	// 正常处理逻辑
	Operate(runtimeObj) error
	// 处理失败后逻辑
	ReconcileFail(runtimeObj, error)
	// 删除逻辑
	IsDeleting(runtimeObj) (bool, error)
}

// reconcile公共逻辑流程
func HandleReconcile(r ISQBReconciler) (ctrl.Result, error) {
	if !entity.ConfigMapData.Initialized {
		return ctrl.Result{RequeueAfter: time.Second}, nil
	}

	obj, err := r.GetInstance()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		} else {
			r.ReconcileFail(obj, err)
			return ctrl.Result{}, err
		}
	}

	if yes, err := r.IsInitialized(obj); !yes {
		if err != nil {
			r.ReconcileFail(obj, err)
		}
		return ctrl.Result{Requeue: true}, nil
	}

	if yes, err := r.IsDeleting(obj); yes {
		if err != nil {
			r.ReconcileFail(obj, err)
		}
		return ctrl.Result{}, err
	}

	err = r.Operate(obj)
	if err != nil {
		r.ReconcileFail(obj, err)
	}

	return ctrl.Result{}, err
}

func CreateOrUpdate(ctx context.Context, obj runtimeObj) error {
	if obj == nil {
		return nil
	}
	if obj.GetCreationTimestamp().Time.IsZero() {
		if err := k8sclient.Create(ctx, obj); err != nil {
			return err
		}
	}
	if err := k8sclient.Update(ctx, obj); err != nil {
		return err
	}
	return nil
}

func Delete(ctx context.Context, obj runtimeObj) error {
	if obj == nil {
		return nil
	}
	if err := k8sclient.Delete(ctx, obj); err != nil {
		return err
	}
	return nil
}

func RemoveFinalizer(ctx context.Context, obj runtimeObj, finalizer string) error {
	controllerutil.RemoveFinalizer(obj, finalizer)
	return k8sclient.Update(ctx, obj)
}
