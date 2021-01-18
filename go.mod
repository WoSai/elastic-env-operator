module github.com/wosai/elastic-env-operator

go 1.13

require (
	github.com/VictoriaMetrics/operator v0.7.1
	github.com/evanphx/json-patch v4.5.0+incompatible
	github.com/go-logr/logr v0.1.0
	github.com/gogo/protobuf v1.3.1
	github.com/imdario/mergo v0.3.9
	github.com/onsi/ginkgo v1.12.1
	github.com/onsi/gomega v1.10.1
	github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring v0.43.0
	github.com/stretchr/testify v1.5.1
	gotest.tools v2.2.0+incompatible
	istio.io/api v0.0.0-20200812202721-24be265d41c3
	istio.io/client-go v0.0.0-20200814134724-bcbf0ed82b30
	k8s.io/api v0.18.6
	k8s.io/apiextensions-apiserver v0.18.6
	k8s.io/apimachinery v0.18.6
	k8s.io/client-go v12.0.0+incompatible
	sigs.k8s.io/controller-runtime v0.6.2
)

replace k8s.io/client-go => k8s.io/client-go v0.18.6
