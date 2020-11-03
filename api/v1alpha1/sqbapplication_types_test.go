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
			Annotations: map[string]string{"a":"1"},
		},
	}
	news := &SQBApplication{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"b":"2"},
		},
	}
	old.Merge(news)
	assert.Equal(t, old.Annotations["a"], "1")
	assert.Equal(t, old.Annotations["b"], "2")
	news = &SQBApplication{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"a":"2"},
		},
	}
	old.Merge(news)
	assert.Equal(t, old.Annotations["a"], "2")
}

func TestHost(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			IngressSpec: IngressSpec{
				Hosts: []string{"1"},
			},
		},
	}
	news := &SQBApplication{
		Spec: SQBApplicationSpec{
			IngressSpec: IngressSpec{
				Hosts: []string{"2"},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, old.Spec.Hosts[0], "1")
	assert.Equal(t, old.Spec.Hosts[1], "2")
	news = &SQBApplication{
		Spec: SQBApplicationSpec{
			IngressSpec: IngressSpec{
				Hosts: []string{"2", "3"},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Hosts), 3)
	assert.Equal(t, old.Spec.Hosts[0], "1")
	assert.Equal(t, old.Spec.Hosts[1], "2")
	assert.Equal(t, old.Spec.Hosts[2], "3")
}

func TestReplicaImage(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Replicas: proto.Int(2),
				Image: "test",
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
						Name: "a",
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
						Name: "b",
						Value: "2",
					},
				},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Env), 2)
}

func TestEnv2(t *testing.T) {
	old := &SQBApplication{
		Spec: SQBApplicationSpec{
			DeploySpec: DeploySpec{
				Env: []v1.EnvVar{
					{
						Name: "a",
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
						Name: "a",
						Value: "2",
					},
				},
			},
		},
	}
	old.Merge(news)
	assert.Equal(t, len(old.Spec.Env), 1)
	assert.Equal(t, old.Spec.Env[0].Value, "2")
}