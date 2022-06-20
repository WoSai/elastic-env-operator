package controllers

import (
	"reflect"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

var (
	// 处理创建删除和generation、annotation的更新
	GenerationAnnotationPredicate = predicate.Funcs{
		UpdateFunc: func(event event.UpdateEvent) bool {
			if event.ObjectOld == nil || event.ObjectNew == nil {
				return false
			}
			//generation不变且annotation也不变，不处理
			if event.ObjectNew.GetGeneration() == event.ObjectOld.GetGeneration() &&
				reflect.DeepEqual(event.ObjectOld.GetAnnotations(), event.ObjectNew.GetAnnotations()) {
				return false
			}
			return true
		},
		GenericFunc: func(event event.GenericEvent) bool {
			return false
		},
	}
)
