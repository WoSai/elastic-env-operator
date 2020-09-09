package v1alpha1

import (
	"github.com/wosai/elastic-env-operator/api/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
)

// 参考：https://github.com/istio/client-go/blob/master/pkg/clientset/versioned/scheme/register.gen.go
var (
	Scheme         = runtime.NewScheme()
	Codecs         = serializer.NewCodecFactory(Scheme)
	ParameterCodec = runtime.NewParameterCodec(Scheme)

	localSchemeBuilder = runtime.SchemeBuilder{
		v1alpha1.AddToScheme,
	}

	AddToScheme = localSchemeBuilder.AddToScheme
)

func init() {
	v1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})
	utilruntime.Must(AddToScheme(Scheme))
}
