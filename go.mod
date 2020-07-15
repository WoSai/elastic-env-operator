module github.com/wosai/elastic-env-operator

go 1.13

require (
	github.com/operator-framework/operator-sdk v0.17.2
	go.uber.org/zap v1.14.1
	istio.io/client-go v0.0.0-20200708142230-d7730fd90478
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.5.2
)

replace (
	github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible // Required by OLM
	k8s.io/client-go => k8s.io/client-go v0.17.4 // Required by prometheus-operator
)
