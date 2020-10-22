package entity

import (
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	appv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SQBPlane struct {
	qav1alpha1.SQBPlane
	Deployments *appv1.DeploymentList
}

func (in *SQBPlane) BuildSelf() {
	if in.Status.Initialized {
		return
	}
	controllerutil.AddFinalizer(in, SqbplaneFinalizer)
	in.Status.Initialized = true
}

