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
	if data["ingressOpen"] == "true" {
		sc.ingressOpen = true
	} else {
		sc.ingressOpen = false
	}
	if data["istioInject"] == "true" {
		sc.istioInject = true
	} else {
		sc.istioInject = false
	}
	if data["istioEnable"] == "true" {
		sc.istioEnable = true
	} else {
		sc.istioEnable = false
	}
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
	return sc.istioInject
}

func (sc *SQBConfigMapEntity) GlobalDeploy() (string, bool) {
	enable := false
	if sc.globalDefaultDeploy != "" {
		enable = true
	}
	return sc.globalDefaultDeploy, enable
}
