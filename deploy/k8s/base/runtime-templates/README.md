# runtime-templates/README.md
# 动态实验环境与 CTF 竞赛环境运行时模板说明

本目录不放固定实例清单，而是沉淀模块04/05 共用的 Kubernetes 运行时模板约定。

## 命名约定

- 实验环境 namespace：`${namespace_prefix}-{instance_id}`
  - `namespace_prefix` 来源于 `backend/configs/config.yaml` 中的 `k8s.namespace_prefix`
  - 当前默认前缀：`lenschain-exp`
- 解题赛 namespace：`ctf-{competition_id}-{team_id}-{challenge_id}`
- 攻防赛 namespace：`ctf-ad-{competition_id}-{group_id}`

## 目录职责

- `resource-quota.template.yaml`：运行时资源配额模板
- `limit-range.template.yaml`：运行时默认资源限制模板
- `network-policy-experiment.template.yaml`：实验环境隔离模板
- `network-policy-ctf.template.yaml`：CTF 环境隔离模板
- `runtime-labels.md`：统一标签/注解约定

## 使用约束

- 模块04/05 的具体实例环境应由后端编排服务动态生成，不应手写为静态固定 YAML。
- 本目录模板的变量由编排服务在创建 namespace / workload 时替换。
- experiment 与 ctf 共享模板模型，但 namespace 规则、权限范围和网络暴露策略可以不同。
