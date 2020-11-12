package entity

const (
	XEnvFlag                     = "x-env-flag"
	AppKey                       = "app"
	PlaneKey                     = "version"
	TeamKey                      = "team"
	GroupKey                     = "group"
	Finalizer                    = "ELASTIC_ENV_OPERATOR"
	ExplicitDeleteAnnotationKey  = "qa.shouqianba.com/delete"
	IstioInjectAnnotationKey     = "qa.shouqianba.com/istio-inject"
	IngressOpenAnnotationKey     = "qa.shouqianba.com/ingress-open"
	PublicEntryAnnotationKey     = "qa.shouqianba.com/public-entry"
	ServiceMonitorAnnotationKey  = "qa.shouqianba.com/service-monitor"
	InitContainerAnnotationKey   = "qa.shouqianba.com/init-container-image"
	DeploymentAnnotationKey      = "qa.shouqianba.com/passthrough-deployment"
	PodAnnotationKey             = "qa.shouqianba.com/passthrough-pod"
	ServiceAnnotationKey         = "qa.shouqianba.com/passthrough-service"
	IngressAnnotationKey         = "qa.shouqianba.com/passthrough-ingress"
	DestinationRuleAnnotationKey = "qa.shouqianba.com/passthrough-destinationrule"
	VirtualServiceAnnotationKey  = "qa.shouqianba.com/passthrough-virtualservice"
	InitializeAnnotationKey      = "qa.shouqianba.com/initialized"
)
