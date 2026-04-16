# runtime-labels.md
# 动态运行时统一标签与注解约定

## 通用标签
- `lenschain.io/runtime-type`
  - `experiment`
  - `ctf-jeopardy`
  - `ctf-attack-defense`
- `lenschain.io/runtime-scope`
  - `dynamic`
- `app.kubernetes.io/part-of=lenschain`

## experiment 扩展标签
- `lenschain.io/instance-id`
- `lenschain.io/template-id`
- `lenschain.io/course-id`
- `lenschain.io/school-id`

## ctf-jeopardy 扩展标签
- `lenschain.io/competition-id`
- `lenschain.io/team-id`
- `lenschain.io/challenge-id`
- `lenschain.io/challenge-type`

## ctf-attack-defense 扩展标签
- `lenschain.io/competition-id`
- `lenschain.io/group-id`
- `lenschain.io/team-id`
- `lenschain.io/runtime-role`
  - `judge-chain`
  - `judge-service`
  - `team-chain`
  - `team-tools`

## 注解建议
- `lenschain.io/created-by=orchestrator`
- `lenschain.io/expire-at=<RFC3339 时间>`
- `lenschain.io/reclaim-policy=destroy|snapshot|archive`
