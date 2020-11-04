package entity

import (
	"k8s.io/apimachinery/pkg/runtime"
)

const (
	XEnvFlag                     = "x-env-flag"
	AppKey                       = "app"
	PlaneKey                     = "version"
	TeamKey                      = "team"
	GroupKey                     = "group"
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
	InitializeAnnotationKey      = "qa.shouqianba.com/initialized"
)

var (
	K8sScheme *runtime.Scheme
)

func SetK8sScheme(s *runtime.Scheme) {
	K8sScheme = s
}
