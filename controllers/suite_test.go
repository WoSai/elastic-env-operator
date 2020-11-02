/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/gogo/protobuf/proto"
	"github.com/wosai/elastic-env-operator/domain/entity"
	"github.com/wosai/elastic-env-operator/domain/handler"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/envtest/printer"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	v1beta13 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecsWithDefaultAndCustomReporters(t,
		"Controller Suite",
		[]Reporter{printer.NewlineReporter{}})
}

var _ = BeforeSuite(func(done Done) {
	logf.SetLogger(zap.LoggerTo(GinkgoWriter, true))

	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "config", "crd", "bases")},
	}

	var err error
	cfg, err = testEnv.Start()
	Expect(err).ToNot(HaveOccurred())
	Expect(cfg).ToNot(BeNil())

	err = scheme.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = qav1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = istio.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1beta1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = v1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme.Scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	handler.SetK8sClient(mgr.GetClient())
	entity.SetK8sScheme(mgr.GetScheme())
	handler.SetK8sLog(ctrl.Log.WithName("domain handler"))

	err = (&SQBDeploymentReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SQBDeployment"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&SQBPlaneReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SQBPlane"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&SQBApplicationReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("SQBApplication"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&ConfigMapReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ConfigMap"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		err = mgr.Start(ctrl.SetupSignalHandler())
		Expect(err).ToNot(HaveOccurred())
	}()
	k8sClient = mgr.GetClient()
	Expect(k8sClient).ToNot(BeNil())

	virtualServiceCRD := &v1beta13.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "virtualservices.networking.istio.io"},
		Spec: v1beta13.CustomResourceDefinitionSpec{
			Group: "networking.istio.io",
			Names: v1beta13.CustomResourceDefinitionNames{
				Plural: "virtualservices",
				Kind:   "VirtualService",
			},
			Scope:                 "Namespaced",
			PreserveUnknownFields: proto.Bool(true),
			Versions: []v1beta13.CustomResourceDefinitionVersion{
				{
					Name:    "v1beta1",
					Served:  true,
					Storage: true,
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), virtualServiceCRD)
	Expect(err).NotTo(HaveOccurred())

	destinationRuleCRD := &v1beta13.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "destinationrules.networking.istio.io"},
		Spec: v1beta13.CustomResourceDefinitionSpec{
			Group: "networking.istio.io",
			Names: v1beta13.CustomResourceDefinitionNames{
				Plural: "destinationrules",
				Kind:   "DestinationRule",
			},
			Scope: "Namespaced",
			Versions: []v1beta13.CustomResourceDefinitionVersion{
				{
					Name:    "v1beta1",
					Served:  true,
					Storage: true,
				},
			},
		},
	}
	err = k8sClient.Create(context.Background(), destinationRuleCRD)
	Expect(err).NotTo(HaveOccurred())

	time.Sleep(time.Second * 5)

	entity.ConfigMapData.FromMap(map[string]string{
		"ingressOpen": "true",
		"istioInject": "true",
		"istioEnable": "true",
	})

	close(done)
}, 60)

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
