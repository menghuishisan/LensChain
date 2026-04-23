// experiment.ts
// 模块04实验配置纯函数：端口冲突、条件环境变量、服务发现变量和结果汇总。

import type { ExperimentInstanceDetail, ImageEnvVarItem } from "@/types/experiment";

/** 容器端口配置输入。 */
export interface ExperimentPortContainer {
  container_name: string;
  ports: Array<{ container: number; protocol: string }>;
}

/** 端口冲突结果。 */
export interface ExperimentPortConflict {
  protocol: string;
  port: number;
  containers: string[];
}

/** 服务发现环境变量。 */
export interface ServiceDiscoveryEnvVar {
  key: string;
  value: string;
}

function normalizeProtocol(protocol: string) {
  return protocol.trim().toLowerCase() || "tcp";
}

function toEnvKeyPart(value: string) {
  const normalized = value.trim().toUpperCase().replace(/[^A-Z0-9]+/g, "_").replace(/^_+|_+$/g, "");
  return normalized.length > 0 ? normalized : "CONTAINER";
}

function readEnv(context: Record<string, string>, key: string, fallback: string) {
  return context[key] ?? fallback;
}

/**
 * detectPortConflicts 检测同协议端口冲突，供模板编排器持续提示。
 */
export function detectPortConflicts(containers: ExperimentPortContainer[]) {
  const groups = new Map<string, { protocol: string; port: number; containers: string[] }>();

  containers.forEach((container) => {
    container.ports.forEach((port) => {
      const protocol = normalizeProtocol(port.protocol);
      const key = `${protocol}:${port.container}`;
      const current = groups.get(key) ?? { protocol, port: port.container, containers: [] };
      current.containers.push(container.container_name);
      groups.set(key, current);
    });
  });

  return Array.from(groups.values())
    .filter((item) => item.containers.length > 1)
    .map<ExperimentPortConflict>((item) => ({ protocol: item.protocol, port: item.port, containers: item.containers }));
}

/**
 * resolveConditionalEnvVars 按镜像配置模板条件规则展开环境变量。
 */
export function resolveConditionalEnvVars(envVars: ImageEnvVarItem[], context: Record<string, string>) {
  const result: ServiceDiscoveryEnvVar[] = [];
  const seenKeys = new Set<string>();

  envVars.forEach((envVar) => {
    const value = readEnv(context, envVar.key, envVar.value);
    result.push({ key: envVar.key, value });
    seenKeys.add(envVar.key);

    envVar.conditions?.forEach((condition) => {
      if (readEnv(context, condition.when, "") !== condition.value) {
        return;
      }
      condition.inject_vars.forEach((item) => {
        if (seenKeys.has(item.key)) {
          return;
        }
        result.push({ key: item.key, value: item.value });
        seenKeys.add(item.key);
      });
    });
  });

  return result;
}

/**
 * buildServiceDiscoveryEnvVars 根据容器名和端口生成服务发现变量，并处理变量名冲突。
 */
export function buildServiceDiscoveryEnvVars(containers: ExperimentPortContainer[]) {
  const result: ServiceDiscoveryEnvVar[] = [];
  const nameCounts = new Map<string, number>();

  containers.forEach((container) => {
    const baseKey = toEnvKeyPart(container.container_name);
    const nextCount = (nameCounts.get(baseKey) ?? 0) + 1;
    nameCounts.set(baseKey, nextCount);
    const keyPrefix = nextCount === 1 ? baseKey : `${baseKey}_${nextCount}`;

    result.push({ key: `${keyPrefix}_HOST`, value: container.container_name });
    container.ports.forEach((port) => {
      const protocol = normalizeProtocol(port.protocol).toUpperCase();
      result.push({ key: `${keyPrefix}_${protocol}_${port.container}`, value: `${container.container_name}:${port.container}` });
    });
  });

  return result;
}

/**
 * buildExperimentResultSummary 汇总实例结果页需要的通过数、检查点得分、总分和通过率。
 */
export function buildExperimentResultSummary(instance: Pick<ExperimentInstanceDetail, "checkpoints" | "scores">) {
  const total = instance.checkpoints.length;
  const passed = instance.checkpoints.filter((checkpoint) => checkpoint.result?.is_passed === true).length;
  const checkpointScore = instance.checkpoints.reduce((sum, checkpoint) => sum + (checkpoint.result?.score ?? 0), 0);
  const totalScore = instance.scores.total_score ?? checkpointScore;
  const passRate = total === 0 ? 0 : Math.round((passed / total) * 100);

  return {
    passed,
    total,
    checkpointScore,
    totalScore,
    passRate,
  };
}
