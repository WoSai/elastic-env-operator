package handler

import (
	"github.com/stretchr/testify/assert"
	appv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"testing"
)

var deployment_handler = &deploymentHandler{}

func TestMerge(t *testing.T) {
	specString := `{
	"replicas": 1, 
	"template": {
		"spec": {
			"dnsConfig": {
				"nameservers": ["1.1.1.1", "2.2.2.2"]
			}
		}
	}
}`

	src := &appv1.Deployment{
		Spec: appv1.DeploymentSpec{
			Template: v1.PodTemplateSpec{
				Spec: v1.PodSpec{
					DNSConfig: &v1.PodDNSConfig{
						Searches: []string{"a", "b"},
					},
				},
			},
		},
	}

	err := deployment_handler.merge(src, specString)
	assert.Nil(t, err)
	assert.Equal(t, src.Spec.Template.Spec.DNSConfig.Nameservers, []string{"1.1.1.1", "2.2.2.2"})
	assert.Equal(t, src.Spec.Template.Spec.DNSConfig.Searches, []string{"a", "b"})
}
