package entity

import (
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SQBPlaneEntity struct {
	qav1alpha1.SQBPlane
}

func NewSQBPlane(namespace, name, description string) *SQBPlaneEntity {
	return &SQBPlaneEntity{
		SQBPlane: qav1alpha1.SQBPlane{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: namespace,
				Name:      name,
			},
			Spec: qav1alpha1.SQBPlaneSpec{
				Description: description,
			},
		},
	}
}

func (in *SQBPlaneEntity) Initialize() {
	controllerutil.AddFinalizer(in, SqbplaneFinalizer)
	if len(in.Annotations) == 0 {
		in.Annotations = make(map[string]string)
	}
	in.Annotations[InitializeAnnotationKey] = "true"
}
