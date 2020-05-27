# elastic-env-operator

![master](https://github.com/WoSai/elastic-env-operator/workflows/master/badge.svg?branch=master) 
![codecov](https://codecov.io/gh/WoSai/elastic-env-operator/branch/master/graph/badge.svg)


**弹性环境**是收钱吧内部基于Kubernetes/Istio实现的集开发、测试、预发布环境于一身的环境，每个开发、测试人员可以在该环境中快速扩展出一套链路闭合、无交叉影响的专属环境。

整体效果类似阿里的特性环境，如下图：

![](https://cdn.ancii.com/article/image/v1/ez/Ju/_m/m_JzueHnNG9dtZ-kWJtDtXuQlTGSxuOADzMMhiO2UACYuHTbZLUD4F972VqqlXugLNwCHTQ5r54fuKH1ONqw939cnN5NncBb0UYUQwKy5us.jpg)

本项目是作为弹性环境2.0版本的核心组件，将原弹性环境平台的核心逻辑以Kubernetes Operator的方式整合进Kubernetes生态之中。

## CRD

### ElasticEnvProject

🍺

#### YAML样例

```yaml
apiVersion: qa.shouqianba.com/v1alpha1
kind: ElasticEnvProject
metadata:
  name: simple-server
  namespace: default
spec:
  image: python:3.7
  resouces:
    limit:
      cpu: 200
      memory: 300
    requests:
      cpu: 50
      memory: 100
  ports:
  - protocol: http
    port: 80
    containerPort: 8080
  healthCheck:
    path: "/"
    port: 8080
  command: python
  args:
  - -m
  - http.server
  - 8080
  - --bind
  - 127.0.0.1
```

### ElasticEnvPlane

🍺

#### YAML样例

```yaml
apiVersion: qa.shouqianba.com/v1alpha1
kind: ElasticEnvPlane
metadata:
  name: staging
  namespace: default
spec:
  purpose: base

```

### ElasticEnvMirror

🍺

#### YAML样例

```yaml
apiVersion: qa.shouqianba.com/v1alpha1
kind: ElasticEnvMirror
metadata:
  name: simple-server
  namespace: default
spec:
  selector:
    project: simple-server
    plane: staging
```