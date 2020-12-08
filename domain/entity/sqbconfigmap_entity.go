package entity

import (
	"encoding/json"
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

var ConfigMapData = &SQBConfigMapEntity{}

// operator相关的业务配置实体
type SQBConfigMapEntity struct {
	ingressOpen                  bool
	istioInject                  bool
	istioEnable                  bool
	serviceMonitorEnable         bool
	domainPostfix                map[string]string // {"ingress class":"host"}
	imagePullSecrets             string
	istioTimeout                 int64
	istioGateways                []string
	specialVirtualServiceIngress string
	deploymentSpec               string
	Initialized                  bool
}

func (sc *SQBConfigMapEntity) FromMap(data map[string]string) {
	if len(data) == 0 {
		data = make(map[string]string)
	}
	sc.ingressOpen = data["ingressOpen"] == "true"
	sc.istioInject = data["istioInject"] == "true"
	sc.istioEnable = data["istioEnable"] == "true"
	sc.serviceMonitorEnable = data["serviceMonitorEnable"] == "true"

	if istioTimeout, ok := data["istioTimeout"]; ok {
		timeout, err := strconv.Atoi(istioTimeout)
		if err == nil {
			sc.istioTimeout = int64(timeout)
		} else {
			sc.istioTimeout = 30
		}
	} else {
		sc.istioTimeout = 30
	}
	if domainPostfix, ok := data["domainPostfix"]; ok {
		domains := make(map[string]string)
		_ = json.Unmarshal([]byte(domainPostfix), &domains)
		sc.domainPostfix = domains
	}
	sc.imagePullSecrets = data["imagePullSecrets"]
	if istioGateways, ok := data["istioGateways"]; ok {
		gateways := make([]string, 0)
		_ = json.Unmarshal([]byte(istioGateways), &gateways)
		sc.istioGateways = gateways
	}
	if len(sc.istioGateways) == 0 {
		sc.istioGateways = []string{"mesh"}
	}
	if specialVirtualServiceIngress, ok := data["specialVirtualServiceIngress"]; ok {
		sc.specialVirtualServiceIngress = specialVirtualServiceIngress
	} else {
		sc.specialVirtualServiceIngress = "nginx"
	}
	sc.deploymentSpec = data["deploymentSpec"]
	sc.Initialized = true
}

func (sc *SQBConfigMapEntity) GetDomainNames(prefix string) map[string]string {
	domains := make(map[string]string)
	for k, v := range sc.domainPostfix {
		domains[k] = strings.ReplaceAll(v, "*", prefix)
	}
	return domains
}

func (sc *SQBConfigMapEntity) GetDomainNameByClass(prefix, class string) string {
	return sc.GetDomainNames(prefix)[class]
}

func (sc *SQBConfigMapEntity) GetImagePullSecrets() []v1.LocalObjectReference {
	imagePullSecrets := make([]v1.LocalObjectReference, 0)
	if secretStr := sc.imagePullSecrets; secretStr != "" {
		secrets := strings.Split(secretStr, ",")
		for _, secret := range secrets {
			imagePullSecrets = append(imagePullSecrets, v1.LocalObjectReference{Name: secret})
		}
	}
	return imagePullSecrets
}

func (sc *SQBConfigMapEntity) IstioTimeout() int64 {
	return sc.istioTimeout
}

func (sc *SQBConfigMapEntity) IstioGateways() []string {
	return sc.istioGateways
}

func (sc *SQBConfigMapEntity) IngressOpen() bool {
	return sc.ingressOpen
}

func (sc *SQBConfigMapEntity) IstioEnable() bool {
	return sc.istioEnable
}

func (sc *SQBConfigMapEntity) IstioInject() bool {
	if sc.istioEnable {
		return sc.istioInject
	} else {
		return false
	}
}

func (sc *SQBConfigMapEntity) IsServiceMonitorEnable() bool {
	return sc.serviceMonitorEnable
}

func (sc *SQBConfigMapEntity) SpecialVirtualServiceIngress() string {
	return sc.specialVirtualServiceIngress
}

func (sc *SQBConfigMapEntity) DeploymentSpec() string {
	return sc.deploymentSpec
}
