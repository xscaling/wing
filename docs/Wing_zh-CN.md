# Wing

## 概念

- ReplicaAutoscaler（以下简称 RA）：定义对于某一可伸缩对象执行的弹性伸缩计划
- Scaler：根据一定配置输出期望实例数的处理机，例如 CPU 使用率、内存使用率、QPS，也可以是一些非转换类的，例如定时弹性。在 ReplicaAutoscaler 中呈现为 `.Spec.Targets`，一个 RA 可以具有多个相同的 Scaler。
- Replicator：整合一个或者多个 Scaler 的输出数据，最终统筹决策最终的期望实例数。

## 架构

![](/docs/assets/wing-architecture.png)

## 实现

Wing 的核心由 Scaler 和 Replicator 组成，并且这两部分为了考虑拓展性设计为可插拔的，用户可以根据自己的需求实现自己的 Scaler 和 Replicator。

### ReplicaAutoscaler 结构说明

```yaml
apiVersion: wing.xscaling.dev/v1
kind: ReplicaAutoscaler
metadata:
  annotations:
    # 定义了两条弹性范围的补丁规则，分别是 2022-12-28 11:00 ~ 2022-12-28 12:10 和 1 0 * * * ~ 1 1 * * *，在上述时刻调整弹性上下限为 10 ~ 20 和 33 ~ 44 个实例
    wing.xscaling.dev/replica-patches: |
      [{"start":"2022-12-28 11:00","end":"2022-12-28 12:10","timezone":"Asia/Shanghai","minReplicas":10,"maxReplicas":20},{"start":"1 0 * * *","end":"1 1 * * *","timezone":"Asia/Shanghai","minReplicas":33,"maxReplicas":44}]
  labels:
    app.kubernetes.io/created-by: wing
  name: hyper
  namespace: matrix
spec:
  # 描述弹性伸缩控制的对象为同 namespace 下名为 hyper 的 Deployment 对象
  scaleTargetRef:
    kind: Deployment
    name: hyper
  # 默认弹性范围是 2 ~ 6 个实例
  maxReplicas: 6
  minReplicas: 2
  # 选择使用 simple Replicator
  replicator: simple
  targets:
    # 启用 CPU 弹性
    - metric: cpu
      settings:
        # 配置默认弹性阈值为 CPU Request 60%
        default:
          utilization: 60
        schedules:
          # 在时区 Asia/Shanghai 的早晨 6~7 点设定弹性阈值为 CPU Request 80%
          - end: 0 7 * * *
            start: 0 6 * * *
            timezone: Asia/Shanghai
            settings:
              utilization: 50
          # 在 2022-12-28 11:00 ~ 2022-12-28 12:10 时刻设定弹性阈值为 CPU Request 50%。并且 dirty_config 并不会生效（不在 CPU Scaler 的配置范围中）
          - end: 2022-12-28 12:10
            start: 2022-12-28 11:00
            timezone: Asia/Shanghai
            settings:
              utilization: 50
              dirty_config: 666
```

### 可插拔 Scaler/Replicator

在实现了 `core/engine.Replicator` 或者 `core/engine.Scaler` 接口后，只需要在 `plugin.conf` 中对应的填写并编译即可将自定义插件集成到 Wing 中。

```
# Directives are registered in the order they should be executed.
#
# Ordering is VERY important. Every plugin will feel the effects of all other
# plugin below (after) them during a request, but they must not care what plugin
# above them are doing.

# How to rebuild with updated plugin configurations: Modify the list below and
# run `go generate && go build`

# The parser takes the input format of:
#
#     <plugin-name>:<package-name>
# Or
#     <plugin-name>:<fully-qualified-package-name>
#
# External plugin example:
# cpu:github.com/xscaling/wing/plugins/scaler_cpu
#
# Local plugin example:
# cpu:scaler_cpu

>>> Scaler
cpu:scaler_cpu
memory:scaler_memory
prometheus:scaler_prometheus

>>> Replicator
simple:replicator_simple
```

当前 Wing 已经实现了一些常用的 Scaler 和 Replicator，用户可以根据自己的需求实现自己的 Scaler 和 Replicator。

Scaler 和 Replicator 的配置可以通过 `--config wing.yaml` 中传入，示例如下

```yaml
workers: 3
plugins:
  cpu:
    utilizationToleration: 0.05
  memory:
    utilizationToleration: 0.05
  prometheus:
    toleration: 0.05
    timeout: 5s
    defaultServer:
      serverAddress: http://prometheus
  # using default config
  simple: {}
```

#### Scaler

- cpu & memory：依赖 [kubernetes-sigs/metrics-server](https://github.com/kubernetes-sigs/metrics-server) 实现 Pod CPU 和内存 Request 使用率弹性。
- prometheus：依赖 [Prometheus](https://prometheus.io/) 实现自定义指标弹性。实现了 Prometheus 指标查询接口的时序库也可以使用

对于 scaler 注册时的插件名称即对应 `.spec.targets[].metric`，举个例子

```yaml
# 意味着使用内存插件，给到一个配置为 utilization: 80 的默认配置
spec:
  targets:
    - metric: memory
      settings:
        default:
          utilization: 80
```

#### Replicator

当前实现了 `simple` Replicator 参考现有的 Kubernetes HPA 实现在所有 Scaler 中取最大值，并在缩容时做减速器。默认的 Replicator 为 `simple`，你也可以在 `spec.replicator` 中为每一个 RA 指定不同的 replicator。

### Panic Mode

为了应对突发流量的场景，我们设计了 Panic Mode。它能够在检测到突发流量时临时调整弹性检查时间间隔，以便更快的响应突发流量。未来还将为其引入感知预扩策略。

```yaml
# 当最终期望实例数大于等于当前实例数的 120% 时，触发 Panic Mode 弹性检查时间间隔由默认 60s 缩短为 15s 并保持 30s。
spec:
  strategy:
    panicWindowSeconds: 30s
    panicThreshold: 1.2
```
