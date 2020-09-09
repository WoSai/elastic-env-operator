package controllers

import (
	"context"
	"github.com/gogo/protobuf/proto"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	v1beta12 "istio.io/client-go/pkg/apis/networking/v1beta1"
	v13 "k8s.io/api/apps/v1"
	v12 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	v1beta13 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	//"github.com/gogo/protobuf/proto"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	//"k8s.io/apimachinery/pkg/api/resource"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

func enableIstio() {
	ctx := context.Background()
	virtualServiceCRD := &v1beta13.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{Name: "virtualservices.networking.istio.io"},
	}
	_, _ = controllerutil.CreateOrUpdate(ctx, k8sClient, virtualServiceCRD, func() error {
		virtualServiceCRD.Spec = v1beta13.CustomResourceDefinitionSpec{
			Group: "networking.istio.io",
			Names: v1beta13.CustomResourceDefinitionNames{
				Plural: "virtualservices",
				Kind:   "VirtualService",
			},
			Scope:                 v1beta13.ResourceScope("Namespaced"),
			PreserveUnknownFields: proto.Bool(true),
			Versions: []v1beta13.CustomResourceDefinitionVersion{
				{
					Name:    "v1beta1",
					Served:  true,
					Storage: true,
				},
			},
		}
		return nil
	})
	destinationRuleCRD := &v1beta13.CustomResourceDefinition{
		ObjectMeta: v1.ObjectMeta{Name: "destinationrules.networking.istio.io"},
	}
	_, _ = controllerutil.CreateOrUpdate(ctx, k8sClient, destinationRuleCRD, func() error {
		destinationRuleCRD.Spec = v1beta13.CustomResourceDefinitionSpec{
			Group: "networking.istio.io",
			Names: v1beta13.CustomResourceDefinitionNames{
				Plural: "destinationrules",
				Kind:   "DestinationRule",
			},
			Scope: v1beta13.ResourceScope("Namespaced"),
			Versions: []v1beta13.CustomResourceDefinitionVersion{
				{
					Name:    "v1beta1",
					Served:  true,
					Storage: true,
				},
			},
		}
		return nil
	})
	time.Sleep(time.Second)
}

var _ = Describe("Controller", func() {
	namespace := "default"
	applicationName := "default-app"
	planeName := "base"
	var deploymentName = getSubsetName(applicationName, planeName)
	image := "busybox"
	ctx := context.Background()
	var err error
	var sqbdeployment *qav1alpha1.SQBDeployment
	var sqbplane *qav1alpha1.SQBPlane
	var sqbapplication *qav1alpha1.SQBApplication

	It("create plane success", func() {
		sqbplane := &qav1alpha1.SQBPlane{
			ObjectMeta: v1.ObjectMeta{
				Namespace: namespace,
				Name:      planeName,
			},
			Spec: qav1alpha1.SQBPlaneSpec{
				Description: "test",
			},
		}
		err = k8sClient.Create(ctx, sqbplane)
		time.Sleep(time.Second)
		Expect(err).NotTo(HaveOccurred())
		instance := &qav1alpha1.SQBPlane{}
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: planeName}, instance)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.Status.Initialized).To(BeTrue())
		Expect(containString(instance.Finalizers, SqbplaneFinalizer)).To(BeTrue())
		err = k8sClient.Delete(ctx, sqbplane)
		Expect(err).NotTo(HaveOccurred())
		time.Sleep(time.Second)
	})

	It("create application success,create base sqbdeployment,create service", func() {
		// create application
		sqbapplication := &qav1alpha1.SQBApplication{
			ObjectMeta: v1.ObjectMeta{
				Namespace: namespace,
				Name:      applicationName,
			},
			Spec: qav1alpha1.SQBApplicationSpec{
				ServiceSpec: qav1alpha1.ServiceSpec{
					Ports: []v12.ServicePort{
						{
							Port:       int32(80),
							TargetPort: intstr.FromInt(8080),
							Protocol:   "HTTP",
						},
					},
				},
				DeploySpec: qav1alpha1.DeploySpec{
					Image: image,
				},
			},
		}
		err = k8sClient.Create(ctx, sqbapplication)
		time.Sleep(time.Second)
		Expect(err).NotTo(HaveOccurred())
		instance := &qav1alpha1.SQBApplication{}
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, instance)
		Expect(instance.Status.Initialized).To(BeTrue())
		Expect(containString(instance.Finalizers, SqbapplicationFinalizer)).To(BeTrue())
		// 会创建base plane和sqbdeployment
		basePlane := &qav1alpha1.SQBPlane{}
		err := k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: planeName}, basePlane)
		Expect(err).NotTo(HaveOccurred())
		sqbdeployment := &qav1alpha1.SQBDeployment{}
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, sqbdeployment)
		Expect(err).NotTo(HaveOccurred())
		Expect(sqbdeployment.Status.Initialized).To(BeTrue())
		Expect(containString(sqbdeployment.Finalizers, SqbdeploymentFinalizer)).To(BeTrue())
		deployment := &v13.Deployment{}
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, deployment)
		Expect(err).NotTo(HaveOccurred())
		// service success
		service := &v12.Service{}
		err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, service)
		Expect(err).NotTo(HaveOccurred())
		Expect(service.Spec.Selector[AppKey]).To(Equal(applicationName))
		Expect(service.Spec.Type, v12.ServiceTypeClusterIP)
		port := service.Spec.Ports[0]
		Expect(port.Name).To(Equal("http-80"))
		Expect(port.Protocol).To(Equal(v12.ProtocolTCP))
		Expect(port.Port).To(Equal(int32(80)))
		Expect(port.TargetPort).To(Equal(intstr.FromInt(8080)))

		_ = k8sClient.Delete(ctx, sqbapplication)
		_ = k8sClient.Delete(ctx, basePlane)
		_ = k8sClient.Delete(ctx, sqbdeployment)
		_ = k8sClient.Delete(ctx, deployment)
		_ = k8sClient.Delete(ctx, service)

		time.Sleep(time.Second)
	})

	Describe("istio disabled", func() {
		BeforeEach(func() {
			// 创建默认的application
			sqbapplication = &qav1alpha1.SQBApplication{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      applicationName,
				},
				Spec: qav1alpha1.SQBApplicationSpec{
					ServiceSpec: qav1alpha1.ServiceSpec{
						Ports: []v12.ServicePort{
							{
								Port:       int32(80),
								TargetPort: intstr.FromInt(8080),
								Protocol:   "HTTP",
							},
						},
					},
					DeploySpec: qav1alpha1.DeploySpec{
						Image: "busybox",
					},
				},
			}
			err = k8sClient.Create(ctx, sqbapplication)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			sqbdeployment = &qav1alpha1.SQBDeployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, sqbdeployment)
			Expect(err).NotTo(HaveOccurred())
			sqbplane = &qav1alpha1.SQBPlane{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: planeName}, sqbplane)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, sqbapplication)
			_ = k8sClient.Delete(ctx, sqbdeployment)
			_ = k8sClient.Delete(ctx, sqbplane)
			// 删除service,ingress,deployment
			service := &v12.Service{ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: applicationName}}
			_ = k8sClient.Delete(ctx, service)
			ingress := &v1beta1.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: applicationName}}
			_ = k8sClient.Delete(ctx, ingress)
			deployment := &v13.Deployment{ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: deploymentName}}
			_ = k8sClient.Delete(ctx, deployment)
			time.Sleep(time.Second)
		})

		It("deployment reconcile success", func() {
			sqbdeployment.Spec = qav1alpha1.SQBDeploymentSpec{
				Selector: qav1alpha1.Selector{
					App:   applicationName,
					Plane: planeName,
				},
				DeploySpec: qav1alpha1.DeploySpec{
					Replicas: proto.Int32(2),
					Resources: &v12.ResourceRequirements{
						Limits: v12.ResourceList{
							v12.ResourceCPU: *resource.NewQuantity(2, resource.DecimalSI),
						},
						Requests: v12.ResourceList{
							v12.ResourceCPU: *resource.NewQuantity(1, resource.DecimalSI),
						},
					},
					Env: []v12.EnvVar{
						{
							Name:  "env1",
							Value: "value1",
						},
					},
					HealthCheck: &v12.Probe{
						Handler: v12.Handler{
							HTTPGet: &v12.HTTPGetAction{
								Port: intstr.FromInt(8080),
								Path: "/healthy",
							},
						},
						InitialDelaySeconds: 10,
						TimeoutSeconds:      10,
						PeriodSeconds:       10,
						SuccessThreshold:    1,
						FailureThreshold:    1,
					},
					NodeAffinity: []qav1alpha1.NodeAffinity{
						{
							Weight: 100,
							Key:    "node",
							Values: []string{"qa"},
						},
					},
					Lifecycle: &qav1alpha1.Lifecycle{
						Init: &qav1alpha1.InitHandler{Exec: &v12.ExecAction{Command: []string{"sleep", "1"}}},
						Lifecycle: v12.Lifecycle{
							PostStart: &v12.Handler{
								HTTPGet: &v12.HTTPGetAction{
									Port: intstr.FromInt(8080),
									Path: "/poststart",
								},
							},
							PreStop: &v12.Handler{
								TCPSocket: &v12.TCPSocketAction{
									Port: intstr.FromInt(8080),
								},
							},
						},
					},
					Volumes: []v12.Volume{
						{
							Name: "volume1",
							VolumeSource: v12.VolumeSource{
								HostPath: &v12.HostPathVolumeSource{Path: "/tmp"},
							},
						},
					},
					VolumeMounts: []v12.VolumeMount{
						{
							Name:      "volume1",
							MountPath: "/tmp",
						},
					},
				},
			}
			err = k8sClient.Update(ctx, sqbdeployment)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)

			deployment := &v13.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Spec.Template.Spec.Containers[0].Image).To(Equal("busybox"))
			Expect(deployment.Labels[AppKey]).To(Equal(applicationName))
			Expect(deployment.Labels[PlaneKey]).To(Equal(planeName))
			Expect(deployment.Spec.Template.Labels[AppKey]).To(Equal(applicationName))
			Expect(deployment.Spec.Template.Labels[PlaneKey]).To(Equal(planeName))
			Expect(deployment.Spec.Selector.MatchLabels[AppKey]).To(Equal(applicationName))
			Expect(deployment.Spec.Replicas).To(Equal(proto.Int32(2)))
			container := deployment.Spec.Template.Spec.Containers[0]
			Expect(container.Resources.Limits.Cpu().String()).To(Equal("2"))
			Expect(container.Resources.Requests.Cpu().String()).To(Equal("1"))
			Expect(container.Env[0].Name).To(Equal("env1"))
			Expect(container.Env[0].Value).To(Equal("value1"))
			Expect(container.LivenessProbe.InitialDelaySeconds).To(Equal(int32(10)))
			Expect(container.LivenessProbe.PeriodSeconds).To(Equal(int32(10)))
			Expect(container.LivenessProbe.HTTPGet.Path).To(Equal("/healthy"))
			Expect(container.LivenessProbe.HTTPGet.Port).To(Equal(intstr.FromInt(8080)))
			Expect(container.ReadinessProbe.InitialDelaySeconds).To(Equal(int32(10)))
			Expect(container.ReadinessProbe.PeriodSeconds).To(Equal(int32(10)))
			Expect(container.ReadinessProbe.HTTPGet.Path).To(Equal("/healthy"))
			Expect(container.ReadinessProbe.HTTPGet.Port).To(Equal(intstr.FromInt(8080)))
			Expect(container.Lifecycle.PostStart.HTTPGet.Port).To(Equal(intstr.FromInt(8080)))
			Expect(container.Lifecycle.PostStart.HTTPGet.Path).To(Equal("/poststart"))
			Expect(container.Lifecycle.PreStop.TCPSocket.Port).To(Equal(intstr.FromInt(8080)))
			initContainer := deployment.Spec.Template.Spec.InitContainers[0]
			Expect(initContainer.Image).To(Equal("busybox"))
			Expect(initContainer.Command).To(Equal([]string{"sleep", "1"}))
			Expect(container.VolumeMounts[0].Name).To(Equal("volume1"))
			Expect(container.VolumeMounts[0].MountPath).To(Equal("/tmp"))
			nodeAffinity := deployment.Spec.Template.Spec.Affinity.NodeAffinity.
				PreferredDuringSchedulingIgnoredDuringExecution[0]
			Expect(nodeAffinity.Preference.MatchExpressions[0].Key).To(Equal("node"))
			Expect(nodeAffinity.Preference.MatchExpressions[0].Values[0]).To(Equal("qa"))
			Expect(nodeAffinity.Weight).To(Equal(int32(100)))
			Expect(deployment.Spec.Template.Spec.Volumes[0].Name).To(Equal("volume1"))
			Expect(deployment.Spec.Template.Spec.Volumes[0].HostPath.Path).To(Equal("/tmp"))
		})

		It("ingress close", func() {
			_ = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, sqbapplication)
			sqbapplication.Annotations = map[string]string{}
			sqbapplication.Annotations[IngressOpenAnnotationKey] = "false"
			err = k8sClient.Update(ctx, sqbapplication)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			ingress := &v1beta1.Ingress{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, ingress)
			Expect(err).To(HaveOccurred())
		})

		It("ingress open", func() {
			_ = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, sqbapplication)
			sqbapplication.Annotations = map[string]string{}
			sqbapplication.Annotations[IngressOpenAnnotationKey] = "true"
			sqbapplication.Spec.Subpaths = []qav1alpha1.Subpath{
				{
					Path:        "/v1",
					ServiceName: "version1",
					ServicePort: 8080,
				},
			}
			err = k8sClient.Update(ctx, sqbapplication)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			ingress := &v1beta1.Ingress{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, ingress)
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress.Spec.Rules[0].Host).To(Equal(applicationName + ".beta.iwosai.com"))
			Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName).To(Equal("version1"))
			Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Path).To(Equal("/v1"))
			Expect(ingress.Spec.Rules[0].HTTP.Paths[1].Backend.ServiceName).To(Equal(applicationName))
			Expect(ingress.Spec.Rules[0].HTTP.Paths[1].Path).To(Equal("/"))
		})

		It("pass deployment annotation,pod annotation", func() {
			sqbdeployment.Annotations = map[string]string{
				DeploymentAnnotationKey: `{"type":"deployment"}`,
				PodAnnotationKey:        `{"type":"pod"}`,
			}
			err = k8sClient.Update(ctx, sqbdeployment)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			deployment := &v13.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, deployment)
			Expect(err).NotTo(HaveOccurred())
			Expect(deployment.Annotations["type"]).To(Equal("deployment"))
			Expect(deployment.Spec.Template.Annotations["type"]).To(Equal("pod"))
		})

		It("pass ingress annotation,service annotation", func() {
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, sqbapplication)
			sqbapplication.Annotations = map[string]string{
				IngressAnnotationKey:     `{"type":"ingress"}`,
				ServiceAnnotationKey:     `{"type":"service"}`,
				IngressOpenAnnotationKey: "true",
			}
			err = k8sClient.Update(ctx, sqbapplication)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			ingress := &v1beta1.Ingress{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, ingress)
			Expect(err).NotTo(HaveOccurred())
			Expect(ingress.Annotations["type"]).To(Equal("ingress"))
			service := &v12.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, service)
			Expect(service.Annotations["type"]).To(Equal("service"))
		})

		It("delete sqbapplication without password", func() {
			// sqbdeployment不删除
			err := k8sClient.Delete(ctx, &qav1alpha1.SQBApplication{ObjectMeta: v1.ObjectMeta{
				Namespace: namespace, Name: applicationName,
			}})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, sqbapplication)
			Expect(err).To(HaveOccurred())
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, sqbdeployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("delete sqbdeployment without password", func() {
			// deployment不删除，sqbapplication的status不变
			err = k8sClient.Delete(ctx, &qav1alpha1.SQBDeployment{
				ObjectMeta: v1.ObjectMeta{Name: deploymentName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, sqbdeployment)
			Expect(err).To(HaveOccurred())
			deployment := &v13.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, deployment)
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, sqbapplication)
			_, ok := sqbapplication.Status.Mirrors[deploymentName]
			Expect(ok).To(BeTrue())
		})

		It("delete sqbplane without password", func() {
			// sqbdeployment不删除
			err = k8sClient.Delete(ctx, &qav1alpha1.SQBPlane{
				ObjectMeta: v1.ObjectMeta{Name: planeName, Namespace: namespace},
			})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: planeName}, sqbplane)
			Expect(err).To(HaveOccurred())
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, sqbdeployment)
			Expect(err).NotTo(HaveOccurred())
		})

		It("delete sqbapplication with password", func() {
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, sqbapplication, func() error {
				sqbapplication.Annotations = map[string]string{
					ExplicitDeleteAnnotationKey: getDeleteCheckSum(sqbapplication),
					IngressOpenAnnotationKey:    "true",
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			err = k8sClient.Delete(ctx, &qav1alpha1.SQBApplication{ObjectMeta: v1.ObjectMeta{
				Namespace: namespace, Name: applicationName,
			}})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(2 * time.Second)
			// sqbapplication，sqbdeployment被删除
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, sqbapplication)
			Expect(err).To(HaveOccurred())
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, sqbdeployment)
			Expect(err).To(HaveOccurred())
			// deployment,ingress和service被删除
			deployment := &v13.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, deployment)
			Expect(err).To(HaveOccurred())
			ingress := &v1beta1.Ingress{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, ingress)
			Expect(err).To(HaveOccurred())
			service := &v12.Service{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, service)
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("istio enabled", func() {
		BeforeEach(func() {
			enableIstio()
			// 创建默认的application
			sqbapplication = &qav1alpha1.SQBApplication{
				ObjectMeta: v1.ObjectMeta{
					Namespace: namespace,
					Name:      applicationName,
				},
				Spec: qav1alpha1.SQBApplicationSpec{
					ServiceSpec: qav1alpha1.ServiceSpec{
						Ports: []v12.ServicePort{
							{
								Port:       int32(80),
								TargetPort: intstr.FromInt(8080),
								Protocol:   "HTTP",
							},
						},
					},
					DeploySpec: qav1alpha1.DeploySpec{
						Image: "busybox",
					},
				},
			}
			err = k8sClient.Create(ctx, sqbapplication)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			sqbdeployment = &qav1alpha1.SQBDeployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, sqbdeployment)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			_ = k8sClient.Delete(ctx, sqbdeployment)
			_ = k8sClient.Delete(ctx, sqbapplication)
			_ = k8sClient.Delete(ctx, sqbplane)
			// 删除service,ingress,deployment,virtualservice,destinationrule
			_ = deleteDeploymentByLabel(k8sClient, ctx, namespace, map[string]string{AppKey: applicationName})
			service := &v12.Service{ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: applicationName}}
			_ = k8sClient.Delete(ctx, service)
			ingress := &v1beta1.Ingress{ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: applicationName}}
			_ = k8sClient.Delete(ctx, ingress)
			virtualservice := &v1beta12.VirtualService{ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: applicationName}}
			_ = k8sClient.Delete(ctx, virtualservice)
			destinationrule := &v1beta12.DestinationRule{ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: applicationName}}
			_ = k8sClient.Delete(ctx, destinationrule)
			time.Sleep(time.Second)
		})

		It("virtualservice created,destinationrule created", func() {
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, sqbapplication, func() error {
				sqbapplication.Annotations = map[string]string{
					IstioInjectAnnotationKey: "true",
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			virtualservice := &v1beta12.VirtualService{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, virtualservice)
			Expect(err).NotTo(HaveOccurred())
			Expect(virtualservice.Spec.Hosts).To(Equal([]string{applicationName + ".beta.iwosai.com",
				applicationName + ".iwosai.com", applicationName}))
			Expect(virtualservice.Spec.Gateways).To(Equal([]string{"mesh"}))
			Expect(virtualservice.Spec.Http[0].Route[0].Destination.Host).To(Equal(applicationName))
			Expect(virtualservice.Spec.Http[0].Route[0].Destination.Subset).To(Equal(deploymentName))
			destinationrule := &v1beta12.DestinationRule{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, destinationrule)
			Expect(err).NotTo(HaveOccurred())
			Expect(destinationrule.Spec.Host).To(Equal(applicationName))
			Expect(destinationrule.Spec.Subsets[0].Name).To(Equal(deploymentName))
			Expect(destinationrule.Spec.Subsets[0].Labels[PlaneKey]).To(Equal(planeName))
		})

		It("ingress open", func() {
			// ingress指向istio-ingressgateway
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, sqbapplication, func() error {
				sqbapplication.Annotations = map[string]string{
					IngressOpenAnnotationKey: "true",
					IstioInjectAnnotationKey: "true",
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			ingress := &v1beta1.Ingress{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, ingress)
			Expect(ingress.Spec.Rules[0].Host).To(Equal(applicationName + ".beta.iwosai.com"))
			Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServiceName).To(Equal("istio-ingressgateway-" + namespace))
			Expect(ingress.Spec.Rules[0].HTTP.Paths[0].Backend.ServicePort).To(Equal(intstr.FromInt(80)))
			Expect(ingress.Spec.Rules[1].Host).To(Equal(applicationName + ".iwosai.com"))
			Expect(ingress.Spec.Rules[1].HTTP.Paths[0].Backend.ServiceName).To(Equal("istio-ingressgateway-" + namespace))
			Expect(ingress.Spec.Rules[1].HTTP.Paths[0].Backend.ServicePort).To(Equal(intstr.FromInt(80)))
		})

		It("public entry", func() {
			//开启特性入口
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, sqbapplication, func() error {
				sqbapplication.Annotations = map[string]string{
					IstioInjectAnnotationKey: "true",
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			_, err = controllerutil.CreateOrUpdate(ctx, k8sClient, sqbdeployment, func() error {
				sqbdeployment.Annotations = map[string]string{
					PublicEntryAnnotationKey: deploymentName + ".iwosai.com",
				}
				return nil
			})
			err = k8sClient.Update(ctx, sqbdeployment)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			// 新增了virtualservice
			virtualservice := &v1beta12.VirtualService{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: deploymentName}, virtualservice)
			Expect(err).NotTo(HaveOccurred())
			Expect(virtualservice.Spec.Hosts).To(Equal([]string{deploymentName + ".iwosai.com"}))
			Expect(virtualservice.Spec.Http[0].Headers.Request.Set[XEnvFlag]).To(Equal(planeName))
		})

		It("pass virtualservice annotation,destinationrule annotation", func() {
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, sqbapplication, func() error {
				sqbapplication.Annotations = map[string]string{
					IstioInjectAnnotationKey:     "true",
					VirtualServiceAnnotationKey:  `{"type":"virtualservice"}`,
					DestinationRuleAnnotationKey: `{"type":"destinationrule"}`,
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			virtualservice := &v1beta12.VirtualService{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, virtualservice)
			Expect(err).NotTo(HaveOccurred())
			Expect(virtualservice.Annotations["type"]).To(Equal("virtualservice"))
			destinationrule := &v1beta12.DestinationRule{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, destinationrule)
			Expect(err).NotTo(HaveOccurred())
			Expect(destinationrule.Annotations["type"]).To(Equal("destinationrule"))
		})

		It("multi subpaths,multi plane", func() {
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, sqbapplication, func() error {
				sqbapplication.Spec.Subpaths = []qav1alpha1.Subpath{
					{
						Path:        "/v2",
						ServiceName: "version2",
						ServicePort: 82,
					},
				}
				sqbapplication.Annotations = map[string]string{
					IstioInjectAnnotationKey: "true",
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			// 加一个sqbplane和sqbdeployment
			plane2 := &qav1alpha1.SQBPlane{
				ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: "test"},
				Spec:       qav1alpha1.SQBPlaneSpec{Description: "test"},
			}
			err = k8sClient.Create(ctx, plane2)
			Expect(err).NotTo(HaveOccurred())
			sqbdeployment2 := &qav1alpha1.SQBDeployment{
				ObjectMeta: v1.ObjectMeta{Namespace: namespace, Name: getSubsetName(applicationName, "test")},
				Spec: qav1alpha1.SQBDeploymentSpec{
					Selector: qav1alpha1.Selector{
						App:   applicationName,
						Plane: "test",
					},
				},
			}
			err = k8sClient.Create(ctx, sqbdeployment2)
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			virtualservice := &v1beta12.VirtualService{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, virtualservice)
			Expect(err).NotTo(HaveOccurred())
			// route顺序为特性环境+subpath，特性环境，基础环境+subpath，基础环境
			httproute0 := virtualservice.Spec.Http[0]
			// match顺序为header,query,sourcelabel
			Expect(len(httproute0.Match)).To(Equal(3))
			Expect(httproute0.Match[0].Headers[XEnvFlag].GetExact()).To(Equal("test"))
			Expect(httproute0.Match[0].Uri.GetPrefix()).To(Equal("/v2"))
			Expect(httproute0.Match[1].QueryParams[XEnvFlag].GetExact()).To(Equal("test"))
			Expect(httproute0.Match[1].Uri.GetPrefix()).To(Equal("/v2"))
			Expect(httproute0.Match[2].SourceLabels[PlaneKey]).To(Equal("test"))
			Expect(httproute0.Match[2].Uri.GetPrefix()).To(Equal("/v2"))
			Expect(httproute0.Route[0].Destination.Host).To(Equal("version2"))
			Expect(httproute0.Route[0].Destination.Subset).To(Equal(getSubsetName("version2", "test")))

			httproute1 := virtualservice.Spec.Http[1]
			Expect(len(httproute1.Match)).To(Equal(3))
			Expect(httproute1.Match[0].Headers[XEnvFlag].GetExact()).To(Equal("test"))
			Expect(httproute1.Match[1].QueryParams[XEnvFlag].GetExact()).To(Equal("test"))
			Expect(httproute1.Match[2].SourceLabels[PlaneKey]).To(Equal("test"))
			Expect(httproute1.Route[0].Destination.Host).To(Equal(applicationName))
			Expect(httproute1.Route[0].Destination.Subset).To(Equal(getSubsetName(applicationName, "test")))

			httproute2 := virtualservice.Spec.Http[2]
			Expect(len(httproute2.Match)).To(Equal(1))
			Expect(httproute2.Match[0].Uri.GetPrefix()).To(Equal("/v2"))
			Expect(httproute2.Route[0].Destination.Host).To(Equal("version2"))
			Expect(httproute2.Route[0].Destination.Subset).To(Equal(getSubsetName("version2", planeName)))

			httproute3 := virtualservice.Spec.Http[3]
			Expect(len(httproute3.Match)).To(Equal(0))
			Expect(httproute3.Route[0].Destination.Host).To(Equal(applicationName))
			Expect(httproute3.Route[0].Destination.Subset).To(Equal(deploymentName))
			// sqbapplication的status正确
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, sqbapplication)
			Expect(err).NotTo(HaveOccurred())
			_, ok := sqbapplication.Status.Planes["base"]
			Expect(ok).To(BeTrue())
			_, ok = sqbapplication.Status.Planes["test"]
		})

		It("delete sqbapplication with password", func() {
			_, err := controllerutil.CreateOrUpdate(ctx, k8sClient, sqbapplication, func() error {
				sqbapplication.Annotations = map[string]string{
					ExplicitDeleteAnnotationKey: getDeleteCheckSum(sqbapplication),
					IstioInjectAnnotationKey:    "true",
				}
				return nil
			})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			err = k8sClient.Delete(ctx, &qav1alpha1.SQBApplication{ObjectMeta: v1.ObjectMeta{
				Namespace: namespace, Name: applicationName,
			}})
			Expect(err).NotTo(HaveOccurred())
			time.Sleep(time.Second)
			virtualservice := &v1beta12.VirtualService{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, virtualservice)
			Expect(err).To(HaveOccurred())
			destinationrule := &v1beta12.DestinationRule{}
			err = k8sClient.Get(ctx, types.NamespacedName{Namespace: namespace, Name: applicationName}, destinationrule)
			Expect(err).To(HaveOccurred())
		})

	})
})
