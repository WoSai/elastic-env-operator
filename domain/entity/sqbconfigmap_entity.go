package entity

import (
	"encoding/json"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"strconv"
	"strings"
	"sync"
)

var ConfigMapData = &SQBConfigMapEntity{}

type (
	configMapData struct {
		ingressOpen                  bool              // 默认是否开启ingress
		istioInject                  bool              // 默认是否启用istio
		istioEnable                  bool              // 集群是否安装istio
		serviceMonitorEnable         bool              // 集群是否安装prometheus
		victoriaMetricsEnable        bool              // 集群是否安装victoria metrics,serviceMonitorEnable和victoriaMetricsEnable互斥
		pvcEnable                    bool              // 集群是否使用PVC
		domainPostfix                map[string]string // 默认的域名后缀{"ingress class":"host"}
		imagePullSecrets             string            // 默认的image pull secret名称
		istioTimeout                 int64             // istio连接超时时间
		istioGateways                []string          // virtualservice应用的gateway
		specialVirtualServiceIngress string            // 特性入口的域名对应的ingress class
		deploymentSpec               string            // 默认的deployment全局配置
		operatorDelay                int               // 启动完成后的延迟时间，主要为了operator重启后不全量reconcile
		initContainerImage           string            // init container镜像
	}
)

// operator相关的业务配置实体
type SQBConfigMapEntity struct {
	data        *configMapData
	mux         sync.RWMutex
	initialized bool // 是否初始化，初始化后才开始接收event，初始化之前的event requeue
	ready       bool // 是否已就绪，就绪后才真正开始处理event，Initialized到Ready状态之间的event直接忽略
}

func (sc *SQBConfigMapEntity) FromMap(data map[string]string) {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	if len(data) == 0 {
		data = make(map[string]string)
	}
	sc.data.ingressOpen = data["ingressOpen"] == "true"
	sc.data.istioInject = data["istioInject"] == "true"
	sc.data.istioEnable = data["istioEnable"] == "true"
	sc.data.serviceMonitorEnable = data["serviceMonitorEnable"] == "true"
	sc.data.victoriaMetricsEnable = data["victoriaMetricsEnable"] == "true"
	if sc.data.serviceMonitorEnable && sc.data.victoriaMetricsEnable {
		sc.data.victoriaMetricsEnable = false
	}
	sc.data.pvcEnable = data["pvcEnable"] == "true"

	if istioTimeout, ok := data["istioTimeout"]; ok {
		timeout, err := strconv.Atoi(istioTimeout)
		if err == nil {
			sc.data.istioTimeout = int64(timeout)
		} else {
			sc.data.istioTimeout = 30
		}
	} else {
		sc.data.istioTimeout = 30
	}
	if domainPostfix, ok := data["domainPostfix"]; ok {
		domains := make(map[string]string)
		_ = json.Unmarshal([]byte(domainPostfix), &domains)
		sc.data.domainPostfix = domains
	}
	sc.data.imagePullSecrets = data["imagePullSecrets"]
	if istioGateways, ok := data["istioGateways"]; ok {
		gateways := make([]string, 0)
		_ = json.Unmarshal([]byte(istioGateways), &gateways)
		sc.data.istioGateways = gateways
	}
	if len(sc.data.istioGateways) == 0 {
		sc.data.istioGateways = []string{"mesh"}
	}
	if specialVirtualServiceIngress, ok := data["specialVirtualServiceIngress"]; ok {
		sc.data.specialVirtualServiceIngress = specialVirtualServiceIngress
	} else {
		sc.data.specialVirtualServiceIngress = "nginx"
	}
	sc.data.deploymentSpec = data["deploymentSpec"]
	operatorDeplay, err := strconv.Atoi(data["operatorDelay"])
	if err != nil {
		operatorDeplay = 30
	}
	sc.data.operatorDelay = operatorDeplay
	sc.data.initContainerImage = data["initContainerImage"]
}

func (sc *SQBConfigMapEntity) ToString() string {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return fmt.Sprintf("ingressOpen: %v, istioInject: %v, istioEnable: %v, serviceMonitorEnable: %v, "+
		"victoriaMetricsEnable: %v, pvcEnable: %v",
		sc.data.ingressOpen, sc.data.istioInject, sc.data.istioEnable, sc.data.serviceMonitorEnable,
		sc.data.victoriaMetricsEnable, sc.data.pvcEnable)
}

func (sc *SQBConfigMapEntity) GetDomainNames(prefix string) map[string]string {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	domains := make(map[string]string)
	for k, v := range sc.data.domainPostfix {
		domains[k] = strings.ReplaceAll(v, "*", prefix)
	}
	return domains
}

func (sc *SQBConfigMapEntity) GetDomainNameByClass(prefix, class string) string {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.GetDomainNames(prefix)[class]
}

func (sc *SQBConfigMapEntity) GetImagePullSecrets() []v1.LocalObjectReference {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	imagePullSecrets := make([]v1.LocalObjectReference, 0)
	if secretStr := sc.data.imagePullSecrets; secretStr != "" {
		secrets := strings.Split(secretStr, ",")
		for _, secret := range secrets {
			imagePullSecrets = append(imagePullSecrets, v1.LocalObjectReference{Name: secret})
		}
	}
	return imagePullSecrets
}

func (sc *SQBConfigMapEntity) IstioTimeout() int64 {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.istioTimeout
}

func (sc *SQBConfigMapEntity) IstioGateways() []string {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.istioGateways
}

func (sc *SQBConfigMapEntity) IngressOpen() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.ingressOpen
}

func (sc *SQBConfigMapEntity) IstioEnable() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.istioEnable
}

func (sc *SQBConfigMapEntity) IstioInject() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	if sc.data.istioEnable {
		return sc.data.istioInject
	} else {
		return false
	}
}

func (sc *SQBConfigMapEntity) IsServiceMonitorEnable() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.serviceMonitorEnable
}

func (sc *SQBConfigMapEntity) IsVictoriaMetricsEnable() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.victoriaMetricsEnable
}

func (sc *SQBConfigMapEntity) IsPVCEnable() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.pvcEnable
}

func (sc *SQBConfigMapEntity) SpecialVirtualServiceIngress() string {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.specialVirtualServiceIngress
}

func (sc *SQBConfigMapEntity) DeploymentSpec() string {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.deploymentSpec
}

func (sc *SQBConfigMapEntity) IsInitialized() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.initialized
}

func (sc *SQBConfigMapEntity) IsReady() bool {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.ready
}

func (sc *SQBConfigMapEntity) SetReady() {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	sc.ready = true
}

func (sc *SQBConfigMapEntity) SetInitialized() {
	sc.mux.Lock()
	defer sc.mux.Unlock()
	sc.initialized = true
}

func (sc *SQBConfigMapEntity) OperatorDelay() int {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.operatorDelay
}

func (sc *SQBConfigMapEntity) InitContainerImage() string {
	sc.mux.RLock()
	defer sc.mux.RUnlock()
	return sc.data.initContainerImage
}
