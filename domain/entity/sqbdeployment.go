package entity

import (
	"encoding/json"
	jsonpatch "github.com/evanphx/json-patch"
	types2 "github.com/gogo/protobuf/types"
	qav1alpha1 "github.com/wosai/elastic-env-operator/api/v1alpha1"
	"github.com/wosai/elastic-env-operator/domain/util"
	istioapi "istio.io/api/networking/v1beta1"
	istio "istio.io/client-go/pkg/apis/networking/v1beta1"
	appv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type SQBDeployment struct {
	qav1alpha1.SQBDeployment
	SqbApplication *SQBApplication
	Deployment *appv1.Deployment
	SpecialVirtualService *istio.VirtualService
}

func (in *SQBDeployment) BuildSelf() {
	if in.Status.Initialized == true {
		return
	}

	controllerutil.AddFinalizer(in, SqbdeploymentFinalizer)
	applicationDeploy, _ := json.Marshal(in.SqbApplication.Spec.DeploySpec)
	deploymentDeploy, _ := json.Marshal(in.Spec.DeploySpec)
	mergeDeploy, _ := jsonpatch.MergePatch(applicationDeploy, deploymentDeploy)
	deploy := qav1alpha1.DeploySpec{}
	if err := json.Unmarshal(mergeDeploy, &deploy); err == nil {
		in.Spec.DeploySpec = deploy
	}

	in.Labels = util.MergeStringMap(in.SqbApplication.Labels, in.Labels)
	in.Labels = util.MergeStringMap(in.Labels, map[string]string{
		AppKey:   in.Spec.Selector.App,
		PlaneKey: in.Spec.Selector.Plane,
	})
	in.Status.Initialized = true
}

func (in *SQBDeployment) BuildRef() {
	in.buildDeployment()
	in.buildSpecialVirtualService()
}

func (in *SQBDeployment) buildDeployment() {
	if in.Deployment == nil {
		return
	}
	deploy := in.Spec.DeploySpec
	container := corev1.Container{
		Name:           in.Name,
		Image:          deploy.Image,
		Env:            deploy.Env,
		VolumeMounts:   deploy.VolumeMounts,
		LivenessProbe:  deploy.HealthCheck,
		ReadinessProbe: deploy.HealthCheck,
		Command:        deploy.Command,
		Args:           deploy.Args,
	}
	if deploy.Resources != nil {
		container.Resources = *deploy.Resources
	}
	if deploy.Lifecycle != nil {
		var lifecycle corev1.Lifecycle
		if poststart := deploy.Lifecycle.PostStart; poststart != nil {
			lifecycle.PostStart = poststart
		}
		if prestop := deploy.Lifecycle.PreStop; prestop != nil {
			lifecycle.PreStop = prestop
		}
		container.Lifecycle = &lifecycle
	}

	in.Deployment.Labels = in.Labels
	in.Deployment.Spec.Replicas = deploy.Replicas
	in.Deployment.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: map[string]string{
			AppKey: in.Spec.Selector.App,
		},
	}
	in.Deployment.Spec.Template.ObjectMeta.Labels = in.Deployment.Labels
	in.Deployment.Spec.Template.Spec.Volumes = deploy.Volumes
	in.Deployment.Spec.Template.Spec.HostAliases = deploy.HostAlias
	in.Deployment.Spec.Template.Spec.Containers = []corev1.Container{container}
	in.Deployment.Spec.Template.Spec.ImagePullSecrets = ConfigMapData.GetImagePullSecrets()

	if anno, ok := in.Annotations[PodAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &in.Deployment.Spec.Template.Annotations)
	} else {
		in.Deployment.Spec.Template.Annotations = nil
	}

	if anno, ok := in.Annotations[DeploymentAnnotationKey]; ok {
		_ = json.Unmarshal([]byte(anno), &in.Deployment.Annotations)
	} else {
		in.Deployment.Annotations = nil
	}
	if len(in.Deployment.Annotations) == 0 {
		in.Deployment.Annotations = make(map[string]string)
	}
	// sqbapplication controller要用到publicEntry
	if publicEntry, ok := in.Annotations[PublicEntryAnnotationKey]; ok {
		in.Deployment.Annotations[PublicEntryAnnotationKey] = publicEntry
	} else {
		delete(in.Deployment.Annotations, PublicEntryAnnotationKey)
	}
	// init lifecycle
	if deploy.Lifecycle != nil && deploy.Lifecycle.Init != nil {
		init := deploy.Lifecycle.Init
		initContainer := corev1.Container{
			Name:         "busybox",
			Image:        "busybox",
			Command:      init.Exec.Command,
			Env:          deploy.Env,
			VolumeMounts: deploy.VolumeMounts,
		}
		in.Deployment.Spec.Template.Spec.InitContainers = []corev1.Container{initContainer}
	}
	// NodeAffinity
	if deploy.NodeAffinity != nil {
		var nodeAffinity []corev1.PreferredSchedulingTerm
		for _, item := range deploy.NodeAffinity {
			nodeAffinity = append(nodeAffinity, corev1.PreferredSchedulingTerm{
				Weight: item.Weight,
				Preference: corev1.NodeSelectorTerm{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      item.Key,
							Operator: item.Operator,
							Values:   item.Values,
						},
					},
				},
			})
		}
		affinity := &corev1.Affinity{
			NodeAffinity: &corev1.NodeAffinity{
				PreferredDuringSchedulingIgnoredDuringExecution: nodeAffinity,
			},
		}
		in.Deployment.Spec.Template.Spec.Affinity = affinity
	}
}

func (in *SQBDeployment) buildSpecialVirtualService() {
	if in.SpecialVirtualService == nil {
		return
	}
	hosts := ConfigMapData.GetDomainNames(in.Name)
	result := hosts[0]
	for _, host := range hosts[1:] {
		if len(host) < len(result) {
			result = host
		}
	}
	virtualserviceHosts := []string{result}
	in.SpecialVirtualService.Spec.Hosts = virtualserviceHosts
	in.SpecialVirtualService.Spec.Gateways = ConfigMapData.IstioGateways()
	in.SpecialVirtualService.Spec.Http = []*istioapi.HTTPRoute{
		{
			Route: []*istioapi.HTTPRouteDestination{
				{Destination: &istioapi.Destination{
					Host:   in.Labels[AppKey],
					Subset: in.Name,
				}},
			},
			Timeout: &types2.Duration{Seconds: ConfigMapData.IstioTimeout()},
			Headers: &istioapi.Headers{
				Request: &istioapi.Headers_HeaderOperations{Set: map[string]string{XEnvFlag: in.Labels[PlaneKey]}},
			},
		},
	}
	// 为了删除Deployment能自动删除SpecialVirtualservice
	_ = controllerutil.SetControllerReference(in.Deployment, in.SpecialVirtualService, k8sscheme)
}
