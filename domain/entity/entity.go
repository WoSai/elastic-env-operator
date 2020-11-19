package entity

const (
	XEnvFlag                     = "x-env-flag"
	AppKey                       = "app"
	PlaneKey                     = "version"
	TeamKey                      = "team"
	GroupKey                     = "group"
	FINALIZER                    = "operator"
	ExplicitDeleteAnnotationKey  = "qa.shouqianba.com/delete"
	IstioInjectAnnotationKey     = "qa.shouqianba.com/istio-inject"
	IngressOpenAnnotationKey     = "qa.shouqianba.com/ingress-open"
	PublicEntryAnnotationKey     = "qa.shouqianba.com/public-entry"
	ServiceMonitorAnnotationKey  = "qa.shouqianba.com/service-monitor"
	InitContainerAnnotationKey   = "qa.shouqianba.com/init-container-image"
	SpecialVirtualServiceIngress = "qa.shouqianba.com/special-virtualservice-ingressclass"
	DeploymentAnnotationKey      = "qa.shouqianba.com/passthrough-deployment"
	PodAnnotationKey             = "qa.shouqianba.com/passthrough-pod"
	ServiceAnnotationKey         = "qa.shouqianba.com/passthrough-service"
	DestinationRuleAnnotationKey = "qa.shouqianba.com/passthrough-destinationrule"
	VirtualServiceAnnotationKey  = "qa.shouqianba.com/passthrough-virtualservice"
	InitializeAnnotationKey      = "qa.shouqianba.com/initialized"
	IngressClassAnnotationKey    = "kubernetes.io/ingress.class"
)
