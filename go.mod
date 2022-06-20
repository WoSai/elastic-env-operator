module github.com/wosai/elastic-env-operator

go 1.13

require (
	github.com/VictoriaMetrics/operator v0.7.1
	github.com/go-logr/logr v0.3.0
	github.com/go-logr/zapr v0.2.0
	github.com/gogo/protobuf v1.3.2
	github.com/google/gnostic v0.0.0-00010101000000-000000000000 // indirect
	github.com/imdario/mergo v0.3.10
	github.com/onsi/ginkgo v1.14.1
	github.com/onsi/gomega v1.10.2
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.43.0
	github.com/stretchr/testify v1.6.1
	go.uber.org/zap v1.15.0
	gotest.tools v2.2.0+incompatible
	istio.io/api v0.0.0-20200812202721-24be265d41c3
	istio.io/client-go v0.0.0-20200814134724-bcbf0ed82b30
	k8s.io/api v0.20.11
	k8s.io/apiextensions-apiserver v0.20.11
	k8s.io/apimachinery v0.20.11
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.8.3
)

replace (
	github.com/google/gnostic => github.com/googleapis/gnostic v0.6.9
	github.com/googleapis/gnostic => github.com/google/gnostic v0.6.9
	k8s.io/client-go => k8s.io/client-go v0.20.11
)
