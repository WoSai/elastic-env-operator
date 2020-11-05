package handler

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/wosai/elastic-env-operator/domain/entity"
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

func SetK8sClient(c client.Client) {
	k8sclient = c
}

func SetK8sLog(l logr.Logger) {
	log = l
}

func SetK8sScheme(s *runtime.Scheme) {
	k8sScheme = s
}

type runtimeObj interface {
	runtime.Object
	metav1.Object
}

type SQBHandler interface {
	GetInstance() (runtimeObj, error)
	IsInitialized(runtimeObj) (bool, error)
	Operate(runtimeObj) error
	ReconcileFail(runtimeObj, error)
	IsDeleting(runtimeObj) (bool, error)
}

func HandleReconcile(r SQBHandler) (ctrl.Result, error) {
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

	log.Info("get instance", "kind", obj.GetObjectKind().GroupVersionKind(),
		"namespace", obj.GetNamespace(), "name", obj.GetName())

	generation := obj.GetGeneration()
	if yes, err := r.IsInitialized(obj); !yes {
		if err != nil {
			r.ReconcileFail(obj, err)
			return ctrl.Result{}, err
		}
		if generation != obj.GetGeneration() {
			return ctrl.Result{}, nil
		}
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

func Delete(ctx context.Context, obj runtimeObj) error {
	kind, _ := apiutil.GVKForObject(obj, k8sScheme)
	err := k8sclient.Delete(ctx, obj)
	log.Info("delete obj", "kind", kind,
		"namespace", obj.GetNamespace(), "name", obj.GetName(), "error", err)
	return client.IgnoreNotFound(err)
}
