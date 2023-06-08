# Prometheus Scaler

![](/docs/assets/prometheus-scaler-architecture.png)

## 配置

| 配置项          | 必须 | 类型   | 默认值                            | 说明                                                                                                                         |
| --------------- | ---- | ------ | --------------------------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| query           | 是   | string | 空                                | Prometheus 查询语句。特别注意数据的有效范围，Wing 只作用于所在集群的可伸缩对象。                                             |
| threshold       | 是   | float  | 空                                | 弹性伸缩判定阈值                                                                                                             |
| failureMode      | 否   | bool   | false                             | 当查询失败时的处理方式（默认为中断弹性），可选有 `FailAsZero` 异常时判定值为 0；`FailAsLastValue` 异常是使用上一次存储的数值，如果没有可用数值则中断弹性。   |
| serverAddress   | 否   | string | Wing 全局设置的 Prometheus Server | 自定义查询 Prometheus 源地址（兼容 Prometheus Query API 即可）                                                               |
| insecureSSL     | 否   | bool   | false                             | 是否跳过 Prometheus Server 的 SSL 验证                                                                                       |
| bearerToken     | 否   | string | 空                                | Prometheus Server Token Auth 的 Bearer Token                                                                                 |
| username        | 否   | string | 空                                | Prometheus Server HTTP Auth 的用户名                                                                                         |
| password        | 否   | string | 空                                | Prometheus Server HTTP Auth 的密码                                                                                           |
