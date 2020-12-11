package entity

import (
	"gotest.tools/assert"
	"testing"
)

func TestConfigMap(t *testing.T) {
	mapdata := map[string]string{
		"ingressOpen":          "true",
		"istioInject":          "true",
		"istioEnable":          "true",
		"serviceMonitorEnable": "true",
		"domainPostfix":        `{"nginx-internal":"*.beta.iwosai.com","nginx":"*.iwosai.com"}`,
		"globalDefaultDeploy":  `{"replicas": 2}`,
		"imagePullSecrets":     "reg-wosai",
		"istioTimeout":         "30",
		"istioGateways":        `["istio-system/ingressgateway","mesh"]`,
	}
	var configmap = &SQBConfigMapEntity{}
	configmap.FromMap(mapdata)

	t.Run("test ingress open", func(t *testing.T) {
		assert.Equal(t, configmap.IngressOpen(), true)
	})

	t.Run("test istio inject", func(t *testing.T) {
		assert.Equal(t, configmap.IstioInject(), true)
	})

	t.Run("test istio enable", func(t *testing.T) {
		assert.Equal(t, configmap.IstioEnable(), true)
	})

	t.Run("test service monitor", func(t *testing.T) {
		assert.Equal(t, configmap.IsServiceMonitorEnable(), true)
	})

	t.Run("domain names", func(t *testing.T) {
		domains := configmap.GetDomainNames("test")
		assert.Equal(t, domains["nginx-internal"], "test.beta.iwosai.com")
		assert.Equal(t, domains["nginx"], "test.iwosai.com")
	})

	t.Run("image pull secrets", func(t *testing.T) {
		secrets := configmap.GetImagePullSecrets()
		assert.Equal(t, len(secrets), 1)
	})

	t.Run("istio timeout", func(t *testing.T) {
		assert.Equal(t, configmap.IstioTimeout(), int64(30))
	})

	t.Run("istio gateways", func(t *testing.T) {
		gateways := configmap.IstioGateways()
		assert.Equal(t, gateways[0], "istio-system/ingressgateway")
		assert.Equal(t, gateways[1], "mesh")
	})

	t.Run("initialized", func(t *testing.T) {
		assert.Equal(t, configmap.Initialized, true)
	})
}
