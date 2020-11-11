package handler

import (
	"context"
	"github.com/go-logr/logr"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/util"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
			return ctrl.Result{}, err
		}
	}

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

// DeleteAllOf 根据namespace和label删除某种类型的所有资源
func DeleteAllOf(ctx context.Context, obj runtimeObj, namespace string, labelMap map[string]string) error {
	err := k8sclient.DeleteAllOf(ctx, obj, &client.DeleteAllOfOptions{
		ListOptions: client.ListOptions{
			Namespace:     namespace,
			LabelSelector: labels.SelectorFromSet(labelMap),
		},
	})
	kind, _ := apiutil.GVKForObject(obj, k8sScheme)
	log.Info("delete all obj", "kind", kind,
		"namespace", namespace, "label", labelMap, "error", err)
	return err
}

func IsExplicitDelete(obj runtimeObj) bool {
	deleteCheckSum, ok := obj.GetAnnotations()[entity.ExplicitDeleteAnnotationKey]
	if obj.GetDeletionTimestamp() != nil && !obj.GetDeletionTimestamp().Time.IsZero() && ok &&
		deleteCheckSum == util.GetDeleteCheckSum(obj.GetName()) &&
		controllerutil.ContainsFinalizer(obj, entity.Finalizer) {
		return true
	}
	return false
}
