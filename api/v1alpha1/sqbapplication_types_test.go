package v1alpha1

import (
	"github.com/gogo/protobuf/proto"
	"gotest.tools/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestAnnotation(t *testing.T) {
	old := &SQBApplication{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"a": "1"},
		},
	}
	news := &SQBApplication{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"b": "2"},
		},
	}
	old.Merge(news)
	assert.Equal(t, old.Annotations["a"], "1")
	assert.Equal(t, old.Annotations["b"], "2")
	news = &SQBApplication{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"a": "2"},
		},
	}
	old.Merge(news)
	assert.Equal(t, old.Annotations["a"], "2")
}

func TestHost(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			IngressSpec: IngressSpec{
				Domains: []Domain{
					{
						Class: "nginx",
						Host:  "test.iwosai.com",
					},
				},
			},
		},
	}
	news := &SQBApplication{
		Spec: SQBApplicationSpec{
			IngressSpec: IngressSpec{
				Domains: []Domain{
					{
						Class: "nginx-internal",
						Host:  "test.beta.iwosai.com",
					},
				},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Domains), 1)
	assert.Equal(t, old.Spec.Domains[0].Class, "nginx-internal")
	assert.Equal(t, old.Spec.Domains[0].Host, "test.beta.iwosai.com")
	news = &SQBApplication{
		Spec: SQBApplicationSpec{
			IngressSpec: IngressSpec{
				Domains: []Domain{
					{
						Class: "nginx-internal",
						Host:  "test2.beta.iwosai.com",
					},
				},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Domains), 1)
	assert.Equal(t, old.Spec.Domains[0].Class, "nginx-internal")
	assert.Equal(t, old.Spec.Domains[0].Host, "test2.beta.iwosai.com")
}

func TestReplicaImage(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Replicas: proto.Int(2),
				Image:    "test",
			},
		},
	}
	news := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Image: "test1",
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, old.Spec.Image, "test1")
	assert.Equal(t, *old.Spec.Replicas, int32(2))
}

func TestEnv(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Env: []v1.EnvVar{
					{
						Name:  "a",
						Value: "1",
					},
				},
			},
		},
	}
	news := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Env: []v1.EnvVar{
					{
						Name:  "b",
						Value: "2",
					},
				},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Env), 1)
	assert.Equal(t, old.Spec.Env[0].Name, "b")
	assert.Equal(t, old.Spec.Env[0].Value, "2")
}

func TestVolume(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Volumes: []*VolumeSpec{
					{
						MountPath: "/path1",
						ConfigMap: "configmap1",
					},
				},
			},
		},
	}

	news := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Volumes: []*VolumeSpec{
					{
						MountPath: "/path2",
						HostPath:  "/host2",
					},
				},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Volumes), 1)
	assert.Equal(t, old.Spec.Volumes[0].MountPath, "/path2")
	assert.Equal(t, old.Spec.Volumes[0].ConfigMap, "")
	assert.Equal(t, old.Spec.Volumes[0].HostPath, "/host2")

}

func TestPorts(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			ServiceSpec: ServiceSpec{
				Ports: []v1.ServicePort{
					{
						Name: "http-80",
						Port: 80,
					},
				},
			},
		},
	}
	news := &SQBApplication{
		Spec: SQBApplicationSpec{
			ServiceSpec: ServiceSpec{
				Ports: []v1.ServicePort{
					{
						Name: "http-8080",
						Port: 8080,
					},
				},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Ports), 1)
	assert.Equal(t, old.Spec.Ports[0].Port, int32(8080))
}
