## 背景
目前k8s服务部署方式为直接调用api创建，环境的复制和迁移需要多套配置文件。现使用k8s operator方式，只需要少量配置文件，就可以迅速复制和迁移环境，还为后续完善operator调和机制打下基础。

## operator简介
k8s operator就是自定义的k8s controller，我们可以自定义资源(CRD)，然后让operator监控这些资源，来代替用户做一些操作如创建Deployment、Service等。

## 技术选型
- Kubernetes API Version: v1
- istio API version: v1beta1
- operator framework version: v1.0.0
- Kubernetes version： 1.16+


## 资源依赖关系
SQBApplicaiton负责操作Ingress、Service、VirtualService、DestinationRule

SQBDeployment负责操作Deployment

SQBDeployment有指向SQBApplication和SQBPlane的owner reference，以便SQBDeployment发生变更后SQBApplicaiton和SQBPlane可以接收到事件

![](http://sqb-qa.oss-cn-hangzhou.aliyuncs.com/crm%2Fresourcedep.jpg)


## 自定义资源CRD
### SQBApplication
与项目相关的配置，Deployment默认会继承这份配置，可以被SQBDeployment中的配置覆盖
```yaml
apiVersion: qa.shouqianba.com/v1alpha1
kind: SQBApplication
metadata:
  name: merchant-enrolment  # 服务名
  namespace: sqb  # 命名空间
  annotations:
    qa.shouqianba.com/istio-inject: "false" # 是否开启istio注入
    qa.shouqianba.com/ingress-open: "false" # 是否打开ingress
    qa.shouqianba.com/service-monitor: "false"
    qa.shouqianba.com/delete: "xxx"  # md5(metadata.name+salt)得到,salt保存在secret,表示明确删除
    qa.shouqianba.com/passthrough-service: # 透传到Service的annotation,下同
    qa.shouqianba.com/passthrough-destinationrule:
    qa.shouqianba.com/passthrough-virtualservice:
spec:
  # ingress相关配置
  ingress:
    subpaths:  # 没有启用istio注入，作用于ingress，启用istio注入，作用于virtualservice
    - path: /v4
      serviceName: sales-system-service
      servicePort: 80
    domain: 
    - class: 
      annotation:
      hosts:  # hosts，默认会配置 服务名+configmap的domainPostfix，可自定义
      - "merchant-enrolment.beta.iwosai.com"
    - class: 
      annotation:
      hosts:  # hosts，默认会配置 服务名+configmap的domainPostfix，可自定义
      - "merchant-enrolment.beta.iwosai.com"
  # service相关配置
  ports:
  - name: http-80  # name命名规则：{istio支持的protocol}-{port}
    port: 80
    targetPort: 8080
    protocol: TCP  # k8s原生protocol
  # deployment相关配置
  replicas: 1  # 可选，副本数，默认1
  image: # 镜像，必选
  command: # 同k8s container的command
  - sh
  args:    # 同k8s container的args
  - ""
  hostAliases: # 同k8s pod的hostalias
  - hostnames:
    - "apollo.shouqianba.com"
    ip: "172.16.16.235"
  resources: # 资源限制
    limits: # 可选
      cpu: ""
      memory: ""
    requests:
      cpu: ""
      memory: ""
  env: # 环境变量全量支持，与k8s原生保持一致，initContainer也使用同样的env
  - name: "envvar"
    value: ""
  - name: "valueFrom"
    valueFrom:
      fieldRef:
        fieldPath: ""
  - name: "valueFrom"
    valueFrom:
      configMapKeyRef:
        name: ""
        key: ""
  - name: "valueFrom"
    valueFrom:
      secretKeyRef:
        name: ""
        key: ""
  healthCheck:  # 健康检查，同时应用到livenessProbe和readinessProbe,exec,httpGet,tcpSocket三选一
    exec:
      command:
      - ""
    httpGet:
      scheme:
      port:
      path:
    tcpSocket:
      port:
    initialDelaySeconds: # 额外参数，可选
    timeoutSeconds:
    periodSeconds:
    successThreshold:
    failureThreshold:
  volumes:  # volume和volumeMounts全量支持，与k8s原生保持一致
  - name: hostPathVolume
    hostPath:
      path: ""
  - name: secretVolume
    secret:
      secretName: ""
  - name: configMapVolume
    configMap:
      name: ""
  - name: PVCVolume
    persistentVolumeClaim:
      claimName: ""
  volumeMounts: # volumeMounts,initContainer与业务container使用同样的volumeMounts
  - name: ""
    mountPath: ""
  nodeAffinity: # 亲和性，只根据node的label选择,key表示node的label key
    required:
    - key: "role"
      operator: "In" # In,NotIn,Exists,DoesNotExist,Gt,Lt
      values:
      - "qa"
      - "crm"
    prefered:
    - weight: 100
      key: "role"
      operator: "In" # In,NotIn,Exists,DoesNotExist,Gt,Lt
      values:
      - "qa"
      - "crm"
  lifecycle:  # lifecycle hook
    init:  # 使用busybox作为init-container执行一条命令，只支持exec，支持env和volumeMounts
      exec:
        command:
        - ""
    postStart: # exec,httpGet,tcpSocket三选一
      exec:
        command:
        - ""
      httpGet:
        scheme:
        port:
        path:
      tcpSocket:
        port:
    preStop: # 与postStart相同
status:
  planes:
    base: 1
    test: 1
  mirrors: 
    merchant-enrolment-base: 1
```


### SQBPlane
表示环境位面，记录环境位面中有多少服务
```yaml
apiVersion: qa.shouqianba.com/v1alpha1
kind: SQBPlane
metadata:
  name: base # 环境名
  namespace: sqb  # 命名空间
  annotations:
    qa.shouqianba.com/delete: "xxx"  # 明确删除
spec:
  description: # 用途说明
status:
  mirrors:
    merchant-enrolment: 1
    sales-system-api: 1
```

### SQBDeployment
与部署相关的配置，确定部署属于哪个项目，哪个环境位面，默认继承SQBApplication中的配置，可以修改。
```yaml
apiVersion: qa.shouqianba.com/v1alpha1
kind: SQBDeployment
metadata:
  name: merchant-enrolment-base # 部署名
  namespace: sqb
  annotations:
    qa.shouqianba.com/delete: "xxx"  # 明确删除
    qa.shouqianba.com/public-entry: "true" #是否开启外网入口，默认不开启
    qa.shouqianba.com/init-container-image: "registry.wosai-inc.com/xxx" # 初始化容器镜像，默认为busybox
    qa.shouqianba.com/passthrough-deployment: # 透传到下游deployment的annotation
    qa.shouqianba.com/passthrough-pod:
spec:
  selector:  # selector创建之后就不可修改，如果要修改则删除sqbdeployment重新创建
    app: "merchant-enrolment"  # 对应的SQBApp的名字，必选
    plane: "base" # 对应的SQBPlane的名字，可选，默认为base
  # 同SQBApplication的deploy配置，覆盖默认配置
  replicas: 1
status:
```

## controller处理逻辑
### Reconsile Cycle
operator公共流程处理逻辑

![](http://sqb-qa.oss-cn-hangzhou.aliyuncs.com/crm%2Freconsilecycle-1.jpg)

### SQBApplication controller
SQBApplication controller处理逻辑

![](http://sqb-qa.oss-cn-hangzhou.aliyuncs.com/crm%2Fsqbapplication.jpg)

### SQBPlane controller
SQBPlane controller处理逻辑

![](http://sqb-qa.oss-cn-hangzhou.aliyuncs.com/crm%2Fsqbplane.jpg)

### SQBDeployment controller
SQBDeployment controller处理逻辑

![](http://sqb-qa.oss-cn-hangzhou.aliyuncs.com/crm%2Fsqbdeployment.jpg)


## operator的全局配置
### configmap
configmap的data不能为空，否则operator不会生效。  
configmap的namespace与manager保持一致，manager的namespace配置在config/manager/manager.yaml,configmap的name需要为operator-configmap  
配置修改后只对之后创建的服务生效。
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: operator-configmap
  namespace: qa
data:
  ingressOpen: "false" # 集群服务默认是否创建ingress
  istioInject: "false" # 集群服务默认是否开启istio注入
  domainPostfix: "*.beta.iwosai.com,*.iwosai.com" # ingressOpen=true时SQBApplication的ingress host默认会配置SQBApplication name + domainPostfix 域名
  globalDefaultDeploy: |   # 存放默认的SQBApplication的deploy的值
    {"key": "value"}
  imagePullSecrets: "reg-wosai"
  istioTimeout: "30" # istio超时时间，单位秒
  istioGateways: "istio-system/ingressgateway,mesh" # istio的virtualservice的gateways配置
  specialVirtualServiceIngress: | 
    ["vpc","vpn","public"]
```

### secret
存放operator使用到的秘钥信息
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: operator-secret
  namespace: qa
data:
  salt: "xxx" # md5的salt
  
```

## 其他