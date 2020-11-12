package entity

import (
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
)

var ConfigMapData = &SQBConfigMapEntity{}

// operator相关的业务配置实体
type SQBConfigMapEntity struct {
	ingressOpen         bool
	istioInject         bool
	istioEnable         bool
	serviceMonitorEnable bool
	domainPostfix       string
	globalDefaultDeploy string
	imagePullSecrets    string
	istioTimeout        int64
	istioGateways       []string
	Initialized         bool
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
		sc.domainPostfix = domainPostfix
	} else {
		sc.domainPostfix = "*.beta.iwosai.com,*.iwosai.com"
	}
	sc.globalDefaultDeploy = data["globalDefaultDeploy"]
	sc.imagePullSecrets = data["imagePullSecrets"]
	if istioGateways, ok := data["istioGateways"]; ok {
		sc.istioGateways = strings.Split(istioGateways, ",")
	} else {
		sc.istioGateways = []string{"mesh"}
	}
	sc.Initialized = true
}

func (sc *SQBConfigMapEntity) GetDomainNames(prefix string) []string {
	hosts := strings.Split(strings.ReplaceAll(sc.domainPostfix, "*", prefix), ",")
	return hosts
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

func (sc *SQBConfigMapEntity) GlobalDeploy() (string, bool) {
	enable := false
	if sc.globalDefaultDeploy != "" {
		enable = true
	}
	return sc.globalDefaultDeploy, enable
}
