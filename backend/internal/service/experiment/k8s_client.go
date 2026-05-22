// k8s_client.go
// 模块04 — 实验环境：K8s 编排服务真实实现
// 基于 k8s.io/client-go 实现 K8sService 接口
// 负责命名空间管理、Pod 部署/删除、容器命令执行、日志获取、资源监控、镜像预拉取

package experiment

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"maps"
	"path"
	"sort"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/client-go/util/homedir"

	"github.com/lenschain/backend/internal/config"
	"github.com/lenschain/backend/internal/pkg/errcode"
)

const defaultCollectorImageTemplate = "registry.lianjing.com/collector-agents/%s:latest"

// k8sClient K8s 编排服务真实实现
type k8sClient struct {
	clientset  *kubernetes.Clientset
	restConfig *rest.Config
	cfg        config.K8sConfig
}

// NewK8sService 创建 K8s 编排服务实例
// 根据配置决定使用集群内 ServiceAccount 还是外部 kubeconfig
func NewK8sService(cfg config.K8sConfig) (K8sService, error) {
	var restCfg *rest.Config
	var err error

	if cfg.InCluster {
		restCfg, err = rest.InClusterConfig()
	} else {
		kubeconfig := cfg.KubeConfigPath
		if kubeconfig == "" {
			if home := homedir.HomeDir(); home != "" {
				kubeconfig = home + "/.kube/config"
			}
		}
		restCfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	if err != nil {
		return nil, fmt.Errorf("初始化 K8s 配置失败: %w", err)
	}

	if cfg.Timeout > 0 {
		restCfg.Timeout = cfg.Timeout
	}

	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("创建 K8s 客户端失败: %w", err)
	}

	return &k8sClient{
		clientset:  clientset,
		restConfig: restCfg,
		cfg:        cfg,
	}, nil
}

// ---------------------------------------------------------------------------
// 命名空间管理
// ---------------------------------------------------------------------------

// CreateNamespace 创建 K8s 命名空间。
// 若传入资源隔离规格，则会同步创建 ResourceQuota 与 LimitRange。
func (k *k8sClient) CreateNamespace(ctx context.Context, name string, labels map[string]string, resourceSpec *NamespaceResourceSpec) error {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
	}
	_, err := k.clientset.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			if err := k.ensureNamespaceRegistryPullSecret(ctx, name); err != nil {
				return err
			}
			return nil
		}
		return fmt.Errorf("%w: %v", errcode.ErrK8sNamespaceCreateFailed, err)
	}
	if err := k.ensureNamespaceRegistryPullSecret(ctx, name); err != nil {
		return err
	}
	if err := k.applyNamespaceResourceIsolation(ctx, name, resourceSpec); err != nil {
		return err
	}
	return nil
}

// ensureNamespaceRegistryPullSecret 将平台命名空间中的镜像拉取 Secret 同步到动态运行时命名空间。
func (k *k8sClient) ensureNamespaceRegistryPullSecret(ctx context.Context, namespace string) error {
	secretName := strings.TrimSpace(k.cfg.ImagePullSecretName)
	if secretName == "" {
		return nil
	}
	platformNamespace := strings.TrimSpace(k.cfg.PlatformNamespace)
	if platformNamespace == "" {
		platformNamespace = "lenschain"
	}

	if _, err := k.clientset.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{}); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("查询运行时命名空间镜像拉取 Secret 失败: %w", err)
	}

	source, err := k.clientset.CoreV1().Secrets(platformNamespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return fmt.Errorf("平台命名空间缺少镜像拉取 Secret %s", secretName)
		}
		return fmt.Errorf("读取平台镜像拉取 Secret 失败: %w", err)
	}

	cloned := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      source.Name,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "lenschain",
			},
		},
		Type: source.Type,
		Data: maps.Clone(source.Data),
	}
	if _, err := k.clientset.CoreV1().Secrets(namespace).Create(ctx, cloned, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("复制镜像拉取 Secret 失败: %w", err)
	}
	return nil
}

// DeleteNamespace 删除 K8s 命名空间
func (k *k8sClient) DeleteNamespace(ctx context.Context, name string) error {
	return k.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
}

// DeletePodsInNamespace 删除命名空间内的所有 Pod，保留其他所有资源（Service /
// NetworkPolicy / PVC / ResourceQuota / LimitRange）。
//
// 暂停实验时调用：仅 Pod 真正消耗节点 CPU / 内存，删除即可释放配额；其它资源都是
// etcd 元数据 + 选择器，不消耗算力，保留它们让恢复时重建的 Pod 自动通过同名 label
// selector 接管 Service 路由 / NetworkPolicy 隔离 / PVC 数据，零额外动作。
//
// 这是修正之前误用 DeleteNamespace 做暂停清理的根因：
//
//  1. namespace 级联删除是异步的，Terminating 阶段未完成时 CreateNamespace 会失败，
//     用户实测点"恢复"立刻看到 `pods is forbidden: ... namespace is being terminated`。
//  2. namespace 级联会把 PVC 一起带走，与文档 §6.3 "按 default_volumes 自动创建 PV"
//     声明的持久语义直接矛盾，pause/resume 等于 100% 丢数据。
//
// Destroy / Submit / Restart 仍走 DeleteNamespace，namespace 级联删除一次性清干净
// 包括 PVC，对应 F-39 "重新开始：从初始状态" 的语义。
func (k *k8sClient) DeletePodsInNamespace(ctx context.Context, namespace string) error {
	if strings.TrimSpace(namespace) == "" {
		return nil
	}
	if err := k.clientset.CoreV1().Pods(namespace).DeleteCollection(ctx, metav1.DeleteOptions{}, metav1.ListOptions{}); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("删除命名空间 %s 内 Pod 失败: %w", namespace, err)
	}
	return nil
}

// ensurePVC 在指定命名空间内幂等创建 PersistentVolumeClaim。
//
// 设计要点：
//   - 名称即 VolumeSpec.Name，与 Pod 中 PersistentVolumeClaim.ClaimName 相同，方便人肉
//     在 kubectl 里追溯；同名 PVC 已存在直接返回（snapshot 恢复 / Pod 重建复用同一 PVC）。
//   - storageClassName 不指定，沿用集群默认 StorageClass：
//       * 本地开发（Docker Desktop / kind / minikube）：自带 hostpath / standard；
//       * 生产 K8s：由集群运维统一规划（NFS / Ceph / Longhorn 见文档 §6.2）。
//   - AccessMode = ReadWriteOnce：实验 Pod 全部固定单副本（RestartPolicyNever，调度到
//     单一节点），多容器跨容器共享靠 Pod 级 volumeMounts 即可，无需 ReadWriteMany。
//   - Labels 透传 Pod 标签（instance_id 等），便于按实验维度排查 / 清理。
func (k *k8sClient) ensurePVC(ctx context.Context, namespace, name, size string, labels map[string]string) error {
	quantity, err := resource.ParseQuantity(size)
	if err != nil {
		return fmt.Errorf("解析存储容量 %q 失败: %w", size, err)
	}
	if _, err := k.clientset.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, name, metav1.GetOptions{}); err == nil {
		return nil
	} else if !apierrors.IsNotFound(err) {
		return fmt.Errorf("查询 PVC 失败: %w", err)
	}
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: quantity,
				},
			},
		},
	}
	if _, err := k.clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{}); err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("创建 PVC 失败: %w", err)
	}
	return nil
}

// applyNamespaceResourceIsolation 为实验命名空间应用资源配额和默认资源限制。
func (k *k8sClient) applyNamespaceResourceIsolation(ctx context.Context, namespace string, spec *NamespaceResourceSpec) error {
	if spec == nil {
		return nil
	}
	if quota := buildNamespaceResourceQuota(namespace, spec); quota != nil {
		if _, err := k.clientset.CoreV1().ResourceQuotas(namespace).Create(ctx, quota, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("创建命名空间资源配额失败: %w", err)
		}
	}
	if limitRange := buildNamespaceLimitRange(namespace, spec); limitRange != nil {
		if _, err := k.clientset.CoreV1().LimitRanges(namespace).Create(ctx, limitRange, metav1.CreateOptions{}); err != nil {
			return fmt.Errorf("创建命名空间限制范围失败: %w", err)
		}
	}
	return nil
}

// buildNamespaceResourceQuota 构建命名空间 ResourceQuota 对象。
func buildNamespaceResourceQuota(namespace string, spec *NamespaceResourceSpec) *corev1.ResourceQuota {
	if spec == nil {
		return nil
	}
	hard := corev1.ResourceList{}
	appendResourceQuantity(hard, corev1.ResourceLimitsCPU, spec.HardCPU)
	appendResourceQuantity(hard, corev1.ResourceRequestsCPU, spec.HardCPU)
	appendResourceQuantity(hard, corev1.ResourceLimitsMemory, spec.HardMemory)
	appendResourceQuantity(hard, corev1.ResourceRequestsMemory, spec.HardMemory)
	appendResourceQuantity(hard, corev1.ResourceLimitsEphemeralStorage, spec.HardStorage)
	appendResourceQuantity(hard, corev1.ResourceRequestsEphemeralStorage, spec.HardStorage)
	if len(hard) == 0 {
		return nil
	}
	return &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace + "-quota",
			Namespace: namespace,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: hard,
		},
	}
}

// buildNamespaceLimitRange 构建命名空间 LimitRange 对象。
func buildNamespaceLimitRange(namespace string, spec *NamespaceResourceSpec) *corev1.LimitRange {
	if spec == nil {
		return nil
	}
	defaults := corev1.ResourceList{}
	appendResourceQuantity(defaults, corev1.ResourceCPU, spec.DefaultContainerCPU)
	appendResourceQuantity(defaults, corev1.ResourceMemory, spec.DefaultContainerMemory)
	appendResourceQuantity(defaults, corev1.ResourceEphemeralStorage, spec.DefaultContainerStorage)
	if len(defaults) == 0 {
		return nil
	}
	defaultRequests := corev1.ResourceList{}
	appendResourceQuantity(defaultRequests, corev1.ResourceCPU, spec.DefaultContainerCPU)
	appendResourceQuantity(defaultRequests, corev1.ResourceMemory, spec.DefaultContainerMemory)
	appendResourceQuantity(defaultRequests, corev1.ResourceEphemeralStorage, spec.DefaultContainerStorage)
	return &corev1.LimitRange{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespace + "-limits",
			Namespace: namespace,
		},
		Spec: corev1.LimitRangeSpec{
			Limits: []corev1.LimitRangeItem{
				{
					Type:           corev1.LimitTypeContainer,
					Default:        defaults,
					DefaultRequest: defaultRequests,
				},
			},
		},
	}
}

// ---------------------------------------------------------------------------
// Pod 管理
// ---------------------------------------------------------------------------

// DeployPod 部署 Pod（含 Service 和可选 NetworkPolicy）。
//
// Pod 内容器路由规则：
//   - cs.IsInitContainer=true → 进 PodSpec.InitContainers（K8s 串行 exec，全部退出 0 后才
//     启动主容器）。用于业务 bootstrap：cryptogen 生成 MSP、configtxgen 生成 channel artifacts
//     等。
//   - cs.IsInitContainer=false（默认）→ 进 PodSpec.Containers（主容器）。
//
// 由 WaitForTCP 自动派生的 wait-for-deps 容器也会被追加到 InitContainers 末尾（业务
// bootstrap 优先跑，依赖等待最后跑），保证主容器启动时既有 MSP 也有外部依赖就绪。
func (k *k8sClient) DeployPod(ctx context.Context, req *DeployPodRequest) (*DeployPodResponse, error) {
	// 构建容器列表
	containers := make([]corev1.Container, 0, len(req.Containers))
	userInitContainers := make([]corev1.Container, 0)
	for _, cs := range req.Containers {
		container := corev1.Container{
			Name:  cs.Name,
			Image: cs.Image,
			// 本地构建/预拉取的镜像直接复用，避免 K8s 对 :latest 标签默认 Always 重拉
			// 触发对占位 registry（如 registry.lianjing.com）的 DNS 查询失败导致 ImagePullBackOff。
			ImagePullPolicy: corev1.PullIfNotPresent,
			// 实验业务容器是第三方官方镜像（postgres/redis/code-server/remix-ide/blockscout 等），
			// 其入口脚本普遍以 root 启动后通过 gosu/su-exec/su 切换到非特权用户，必须依赖
			// Docker 默认能力集中的 CHOWN/DAC_OVERRIDE/FOWNER/FSETID/SETUID/SETGID/SETPCAP/
			// NET_BIND_SERVICE/KILL 等基础能力。若 Drop:["ALL"]，所有这类镜像在 setuid 第一步
			// 就会以 ExitCode 1 / Pod phase=Failed 立刻退出（详见 docs/modules/04-实验环境/
			// 07-实验类型与环境配置.md §6 容器安全约束）。
			//
			// 生产策略：对齐 K8s PodSecurity "Baseline" 配置——保留 Docker 默认能力集，仅
			// 显式 Drop 危险能力（内核操控、抓包、改时间等），同时通过 AllowPrivilegeEscalation
			// =false + Privileged=false 锁死特权升级路径，再叠加 PodSpec 的 seccomp
			// RuntimeDefault 完成 syscall 过滤。CTF 沙箱如需更严，可在 CTF 运行时按需进一步
			// 收紧而不是改动通用实验路径。
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: boolPtr(false),
				Privileged:               boolPtr(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"SYS_ADMIN",
						"SYS_MODULE",
						"SYS_RAWIO",
						"SYS_PTRACE",
						"SYS_BOOT",
						"SYS_TIME",
						"NET_ADMIN",
						"NET_RAW",
						"MAC_ADMIN",
						"MAC_OVERRIDE",
					},
				},
			},
		}

		// 端口
		for _, p := range cs.Ports {
			proto := corev1.ProtocolTCP
			if strings.EqualFold(p.Protocol, "udp") {
				proto = corev1.ProtocolUDP
			}
			container.Ports = append(container.Ports, corev1.ContainerPort{
				ContainerPort: int32(p.ContainerPort),
				Protocol:      proto,
			})
		}

		// 环境变量
		for envKey, envVal := range cs.EnvVars {
			container.Env = append(container.Env, corev1.EnvVar{
				Name:  envKey,
				Value: envVal,
			})
		}

		// 卷挂载
		for _, v := range cs.Volumes {
			container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
				Name:      v.Name,
				MountPath: v.MountPath,
				SubPath:   v.SubPath,
				ReadOnly:  v.ReadOnly,
			})
		}

		// 资源限制
		cpuLimit := cs.CPULimit
		memLimit := cs.MemoryLimit
		if cpuLimit == "" {
			cpuLimit = k.cfg.DefaultCPU
		}
		if memLimit == "" {
			memLimit = k.cfg.DefaultMemory
		}
		container.Resources = corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuLimit),
				corev1.ResourceMemory: resource.MustParse(memLimit),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse(cpuLimit),
				corev1.ResourceMemory: resource.MustParse(memLimit),
			},
		}

		// 命令和参数
		if len(cs.Command) > 0 {
			container.Command = cs.Command
		}
		if len(cs.Args) > 0 {
			container.Args = cs.Args
		}

		if cs.IsInitContainer {
			userInitContainers = append(userInitContainers, container)
		} else {
			containers = append(containers, container)
		}
	}

	if req.Collector != nil {
		containers = append(containers, k.buildCollectorContainer(req.Collector))
	}

	// 应用层依赖等待（initContainer 模式）。
	//
	// 主容器启动前，跑一个 busybox initContainer 通过 `nc -z` 阻塞到所有依赖
	// host:port 都已 listen。这覆盖了"K8s 报 Pod Running 但应用还在 initdb /
	// 启动 BEAM VM / cryptogen 中"的间隙——纯 K8s 调度顺序无法等到应用真正
	// 就绪，会让 blockscout/CouchDB 等下游容器连接被拒后立即退出，
	// phase=Succeeded 看起来"正常完成"实则启动失败。
	//
	// 用 busybox:1.36（标准最小镜像 < 5MB），imagePullPolicy=IfNotPresent 复用集群缓存。
	// 整个 wait 命令是 `for tgt in ...; do until nc -z host port; do sleep 1; done; done`，
	// 单 init 进程串行等所有依赖，与 docker-compose 的 depends_on 健康检查语义对齐。
	//
	// 顺序：user bootstrap init 先跑（cryptogen / configtxgen），wait-for-deps 后跑
	// （等外部依赖 TCP 就绪）。这样 MSP 在主容器启动前就位，依赖就绪也得到验证。
	initContainers := append([]corev1.Container{}, userInitContainers...)
	initContainers = append(initContainers, buildWaitForTCPInitContainers(req.Containers)...)

	// 构建 Pod 卷。
	//
	// 卷类型规则（详见 VolumeSpec 注释，与文档 09-部署与运维/02 §6.3 一致）：
	//   - Size 非空：image.default_volumes 声明的持久数据，提前建 PVC（按 size 容量）
	//     并在 Pod 中以 PersistentVolumeClaim 引用；同名 VolumeSpec 共用一份 PVC，
	//     允许 Pod 内多容器（init / 主容器）跨容器共享。Pod 重启 / 节点漂移后数据
	//     保留；instance 销毁时由 namespace 级联删除统一回收 PVC，无需手动清理。
	//   - Size 空：临时 Pod 内共享卷，使用 emptyDir，Pod 销毁即回收。
	//
	// PVC 不指定 storageClassName，沿用集群默认 StorageClass（dev/prod 都自带）。
	volumes := make([]corev1.Volume, 0)
	volumeNameSet := make(map[string]bool)
	for _, cs := range req.Containers {
		for _, v := range cs.Volumes {
			if volumeNameSet[v.Name] {
				continue
			}
			volumeNameSet[v.Name] = true
			if v.Size != "" {
				if err := k.ensurePVC(ctx, req.Namespace, v.Name, v.Size, req.Labels); err != nil {
					return nil, fmt.Errorf("创建 PVC %s 失败: %w", v.Name, err)
				}
				volumes = append(volumes, corev1.Volume{
					Name: v.Name,
					VolumeSource: corev1.VolumeSource{
						PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
							ClaimName: v.Name,
							ReadOnly:  v.ReadOnly,
						},
					},
				})
			} else {
				volumes = append(volumes, corev1.Volume{
					Name: v.Name,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				})
			}
		}
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.PodName,
			Namespace: req.Namespace,
			Labels:    req.Labels,
		},
		Spec: corev1.PodSpec{
			InitContainers: initContainers,
			Containers:    containers,
			Volumes:       volumes,
			RestartPolicy: corev1.RestartPolicyNever,
			// EnableServiceLinks=false 关闭 K8s 默认的 Service 环境变量注入。
			//
			// 默认行为下，K8s 会把同 namespace 的每个 Service 以
			// `<NAME>_HOST` / `<NAME>_PORT` / `<NAME>_PORT_<port>_TCP_*` 等环境变量
			// 注入容器，导致 CLI 参数与 env 绑定的二进制（如 geth 把 --port 绑定到
			// $GETH_PORT）解析"tcp://10.x.x.x:8545"为 int 失败而崩溃。
			//
			// 实验 namespace 内的服务发现仅依赖 K8s DNS（service-name.namespace.svc），
			// 不需要 env 注入。关闭后 DNS 解析完全不受影响，但消除了所有此类污染。
			EnableServiceLinks: boolPtr(false),
			SecurityContext: &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
		},
	}
	if secretName := strings.TrimSpace(k.cfg.ImagePullSecretName); secretName != "" {
		pod.Spec.ImagePullSecrets = append(pod.Spec.ImagePullSecrets, corev1.LocalObjectReference{Name: secretName})
	}

	createdPod, err := k.clientset.CoreV1().Pods(req.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errcode.ErrK8sDeployFailed, err)
	}

	// 为有端口的容器创建 Service，服务名与容器名保持一致，和文档里的服务发现变量语义对齐。
	for _, cs := range req.Containers {
		var servicePorts []corev1.ServicePort
		for _, p := range cs.Ports {
			svcPort := p.ServicePort
			if svcPort == 0 {
				svcPort = p.ContainerPort
			}
			proto := corev1.ProtocolTCP
			if strings.EqualFold(p.Protocol, "udp") {
				proto = corev1.ProtocolUDP
			}
			servicePorts = append(servicePorts, corev1.ServicePort{
				Name:       fmt.Sprintf("%s-%d", cs.Name, p.ContainerPort),
				Port:       int32(svcPort),
				TargetPort: intstr.FromInt32(int32(p.ContainerPort)),
				Protocol:   proto,
			})
		}
		if len(servicePorts) == 0 {
			continue
		}
		serviceLabels := maps.Clone(req.Labels)
		serviceLabels["pod-name"] = req.PodName
		svc := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cs.Name,
				Namespace: req.Namespace,
				Labels:    serviceLabels,
			},
			Spec: corev1.ServiceSpec{
				Selector: req.Labels,
				Ports:    servicePorts,
				Type:     corev1.ServiceTypeClusterIP,
			},
		}
		_, _ = k.clientset.CoreV1().Services(req.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	}

	// 可选 NetworkPolicy
	if req.NetworkPolicy != nil {
		_ = k.applyNetworkPolicy(ctx, req)
	}

	// 等待 Pod 进入 Running 阶段并被 kubelet 分配 PodIP 后再返回。
	// Pods().Create() 返回时 Pod 尚未被调度到节点，Status.PodIP 必为空字符串；
	// 若不等待就把空 IP 写入 instance_containers.internal_ip，将导致后续
	// 终端 PTY 代理 / 检查点脚本 / 服务发现等所有需要容器 IP 的链路全部失败。
	readyPod, waitErr := k.waitForPodRunning(ctx, req.Namespace, createdPod.Name)
	if waitErr != nil {
		return nil, fmt.Errorf("%w: 等待 Pod %s 就绪失败: %v", errcode.ErrK8sDeployFailed, createdPod.Name, waitErr)
	}

	return &DeployPodResponse{
		PodName:    readyPod.Name,
		Namespace:  readyPod.Namespace,
		InternalIP: readyPod.Status.PodIP,
		Status:     string(readyPod.Status.Phase),
	}, nil
}

// waitForPodRunning 轮询等待 Pod 进入 Running 阶段且 PodIP 已被分配。
// 超时上限 120 秒（覆盖镜像拉取 + 容器启动），轮询间隔 1 秒。
// 容器进入 Failed/Succeeded 阶段或镜像拉取明确失败时立即返回错误，避免长时间空等。
func (k *k8sClient) waitForPodRunning(ctx context.Context, namespace, podName string) (*corev1.Pod, error) {
	const (
		pollInterval = 1 * time.Second
		pollTimeout  = 120 * time.Second
	)
	deadline := time.Now().Add(pollTimeout)
	for {
		pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		switch pod.Status.Phase {
		case corev1.PodRunning:
			if pod.Status.PodIP != "" {
				return pod, nil
			}
		case corev1.PodFailed, corev1.PodSucceeded:
			return nil, fmt.Errorf("Pod 进入终止状态 phase=%s reason=%s message=%s", pod.Status.Phase, pod.Status.Reason, pod.Status.Message)
		}
		// 检测明确的镜像拉取失败，避免等到超时
		for _, cs := range pod.Status.ContainerStatuses {
			if cs.State.Waiting != nil {
				reason := cs.State.Waiting.Reason
				if reason == "ImagePullBackOff" || reason == "ErrImagePull" || reason == "InvalidImageName" {
					return nil, fmt.Errorf("容器 %s 镜像拉取失败: %s - %s", cs.Name, reason, cs.State.Waiting.Message)
				}
			}
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("等待 Pod 就绪超时 %s（当前 phase=%s）", pollTimeout, pod.Status.Phase)
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
		}
	}
}

// applyNetworkPolicy 应用网络策略
func (k *k8sClient) applyNetworkPolicy(ctx context.Context, req *DeployPodRequest) error {
	np := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.PodName + "-netpol",
			Namespace: req.Namespace,
		},
		Spec: networkingv1.NetworkPolicySpec{
			PodSelector: metav1.LabelSelector{
				MatchLabels: req.Labels,
			},
			PolicyTypes: []networkingv1.PolicyType{
				networkingv1.PolicyTypeIngress,
				networkingv1.PolicyTypeEgress,
			},
		},
	}

	// Ingress 规则
	if len(req.NetworkPolicy.AllowIngress) > 0 || req.NetworkPolicy.AllowSameNamespace || len(req.NetworkPolicy.AllowNamespaceLabelSelectors) > 0 {
		var ingressRules []networkingv1.NetworkPolicyIngressRule
		namespacePeers := buildNamespacePeers(req.Namespace, req.NetworkPolicy)
		if len(namespacePeers) > 0 {
			ingressRules = append(ingressRules, networkingv1.NetworkPolicyIngressRule{
				From: namespacePeers,
			})
		}
		for _, cidr := range req.NetworkPolicy.AllowIngress {
			ingressRules = append(ingressRules, networkingv1.NetworkPolicyIngressRule{
				From: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: &networkingv1.IPBlock{CIDR: cidr},
					},
				},
			})
		}
		np.Spec.Ingress = ingressRules
	}

	// Egress 规则
	if len(req.NetworkPolicy.AllowEgress) > 0 || req.NetworkPolicy.AllowDNS || req.NetworkPolicy.AllowSameNamespace || len(req.NetworkPolicy.AllowNamespaceLabelSelectors) > 0 {
		var egressRules []networkingv1.NetworkPolicyEgressRule
		namespacePeers := buildNamespacePeers(req.Namespace, req.NetworkPolicy)
		if len(namespacePeers) > 0 {
			egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
				To: namespacePeers,
			})
		}
		if req.NetworkPolicy.AllowDNS {
			dnsPortUDP := intstr.FromInt(53)
			dnsPortTCP := intstr.FromInt(53)
			egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
				To: []networkingv1.NetworkPolicyPeer{
					{
						NamespaceSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"kubernetes.io/metadata.name": "kube-system",
							},
						},
						PodSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"k8s-app": "kube-dns",
							},
						},
					},
				},
				Ports: []networkingv1.NetworkPolicyPort{
					{Port: &dnsPortUDP, Protocol: protocolPtr(corev1.ProtocolUDP)},
					{Port: &dnsPortTCP, Protocol: protocolPtr(corev1.ProtocolTCP)},
				},
			})
		}
		for _, cidr := range req.NetworkPolicy.AllowEgress {
			egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
				To: []networkingv1.NetworkPolicyPeer{
					{
						IPBlock: &networkingv1.IPBlock{CIDR: cidr},
					},
				},
			})
		}
		np.Spec.Egress = egressRules
	}

	_, err := k.clientset.NetworkingV1().NetworkPolicies(req.Namespace).Create(ctx, np, metav1.CreateOptions{})
	return err
}

// buildNamespacePeers 构建 NetworkPolicy 的命名空间选择器规则。
func buildNamespacePeers(namespace string, policy *NetworkPolicySpec) []networkingv1.NetworkPolicyPeer {
	if policy == nil {
		return nil
	}
	peers := make([]networkingv1.NetworkPolicyPeer, 0, 1+len(policy.AllowNamespaceLabelSelectors))
	if policy.AllowSameNamespace {
		peers = append(peers, networkingv1.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"kubernetes.io/metadata.name": namespace,
				},
			},
		})
	}
	for _, labels := range policy.AllowNamespaceLabelSelectors {
		if len(labels) == 0 {
			continue
		}
		peers = append(peers, networkingv1.NetworkPolicyPeer{
			NamespaceSelector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
		})
	}
	return peers
}

// DeletePod 删除 Pod 及关联 Service
func (k *k8sClient) DeletePod(ctx context.Context, namespace, podName string) error {
	// 删除关联 Service（忽略不存在错误）
	services, err := k.clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "pod-name=" + podName,
	})
	if err == nil {
		for _, service := range services.Items {
			_ = k.clientset.CoreV1().Services(namespace).Delete(ctx, service.Name, metav1.DeleteOptions{})
		}
	}
	// 删除关联 NetworkPolicy（忽略不存在错误）
	_ = k.clientset.NetworkingV1().NetworkPolicies(namespace).Delete(ctx, podName+"-netpol", metav1.DeleteOptions{})
	// 删除 Pod
	return k.clientset.CoreV1().Pods(namespace).Delete(ctx, podName, metav1.DeleteOptions{})
}

// buildCollectorContainer 构建混合实验所需的 Collector Agent sidecar 容器。
func (k *k8sClient) buildCollectorContainer(spec *CollectorSidecarSpec) corev1.Container {
	imageTemplate := strings.TrimSpace(k.cfg.CollectorImageTemplate)
	if imageTemplate == "" {
		imageTemplate = defaultCollectorImageTemplate
	}
	return corev1.Container{
		Name:            "collector-agent",
		Image:           fmt.Sprintf(imageTemplate, spec.Ecosystem),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Env: []corev1.EnvVar{
			{Name: "COLLECTOR_TARGET_CONTAINER", Value: spec.TargetContainer},
			{Name: "COLLECTOR_ECOSYSTEM", Value: spec.Ecosystem},
			{Name: "COLLECTOR_CONFIG_JSON", Value: string(spec.DataSourceConfig)},
			{Name: "COLLECTOR_SESSION_ID", Value: spec.SessionID},
			{Name: "COLLECTOR_CORE_WS_URL", Value: spec.CoreWebSocketURL},
		},
		Resources: corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("200m"),
				corev1.ResourceMemory: resource.MustParse("256Mi"),
			},
		},
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: boolPtr(false),
			Privileged:               boolPtr(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
		},
	}
}

// GetPodStatus 获取 Pod 状态
func (k *k8sClient) GetPodStatus(ctx context.Context, namespace, podName string) (*PodStatus, error) {
	pod, err := k.clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return k.podToPodStatus(pod), nil
}

// ListPods 列出指定命名空间和标签的 Pod
func (k *k8sClient) ListPods(ctx context.Context, namespace string, labels map[string]string) ([]*PodStatus, error) {
	labelSelector := buildLabelSelector(labels)
	podList, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	result := make([]*PodStatus, 0, len(podList.Items))
	for i := range podList.Items {
		result = append(result, k.podToPodStatus(&podList.Items[i]))
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 容器操作
// ---------------------------------------------------------------------------

// ExecInPod 在 Pod 容器中执行命令
func (k *k8sClient) ExecInPod(ctx context.Context, namespace, podName, container, command string) (*ExecResult, error) {
	return k.execInPod(ctx, namespace, podName, container, []string{"/bin/sh", "-c", command}, nil)
}

// ExecPodPTY 在 Pod 容器内启动 PTY 进程，把 stdin/stdout/resize 桥接到调用方。
//
// 实现要点：
//   - 使用 K8s Pod exec subresource（POST .../exec?stdin&stdout&stderr&tty&container=...），
//     这是行业标准的容器交互通道（kubectl exec / Lens / Rancher / OpenShift Web Terminal
//     全部基于该路径）。NetworkPolicy / Pod 端口 / Service / Ingress 完全不参与，权限仅
//     来自 backend 的 ServiceAccount，业务边界已在 service 层完成。
//   - Stderr 合并到 Stdout：TTY 模式下 K8s 本身就只用一条流（PTY 没有独立 stderr 概念），
//     拆开反而打乱 ANSI 转义。
//   - TerminalSizeQueue 由匿名实现包装调用方传入的 <-chan TerminalSize，channel 关闭时
//     返回 nil 让 client-go 自然停止下发 SIGWINCH，不 panic。
func (k *k8sClient) ExecPodPTY(ctx context.Context, req *ExecPodPTYRequest) error {
	if req == nil {
		return fmt.Errorf("exec pty request is nil")
	}
	if req.Stdin == nil || req.Stdout == nil {
		return fmt.Errorf("exec pty stdin/stdout must not be nil")
	}
	command := req.Command
	if len(command) == 0 {
		// 默认 PTY shell 选取策略（可被 ExecPodPTYRequest.Command 显式覆盖）：
		//
		//   1) 镜像里有 bash —— 拉起 `bash -l`（登录交互 shell），它会 source
		//      /etc/profile + ~/.bashrc，标准 Debian / Ubuntu 系镜像（code-server 的
		//      coder 用户、jupyter 的 jovyan、postgres / fabric-tools / dapp-dev）
		//      这条路径会得到 `\u@\h:\w\$ ` 的提示符，cwd 实时显示。
		//
		//   2) 镜像里没有 bash（典型：alpine 系的 geth / redis / xterm-server）——
		//      显式注入 PS1，使用 `$(whoami)@$(hostname):$PWD$ `。POSIX sh 在每次
		//      渲染提示符前会重新做参数与命令替换，所以 cwd 切换后下一行提示符
		//      自动跟随，不需要 PROMPT_COMMAND。
		//
		// 这个默认值只影响"未显式指定 Command"的场景，service 层调用 ExecPodPTY
		// 仍可以传入业务专用命令（如 `geth attach`、`psql -U ...`）走容器原生 CLI。
		command = []string{"sh", "-c", `if command -v bash >/dev/null 2>&1; then exec bash -l; else PS1='$(whoami)@$(hostname):$PWD$ '; export PS1; exec /bin/sh; fi`}
	}

	execReq := k.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(req.PodName).
		Namespace(req.Namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: req.Container,
			Command:   command,
			Stdin:     true,
			Stdout:    true,
			Stderr:    false,
			TTY:       true,
		}, scheme.ParameterCodec)

	executor, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", execReq.URL())
	if err != nil {
		return fmt.Errorf("创建 PTY 执行器失败: %w", err)
	}

	streamOpts := remotecommand.StreamOptions{
		Stdin:             req.Stdin,
		Stdout:            req.Stdout,
		Tty:               true,
		TerminalSizeQueue: ptySizeQueue{ch: req.Resize},
	}
	if err := executor.StreamWithContext(ctx, streamOpts); err != nil {
		return fmt.Errorf("PTY 流执行失败: %w", err)
	}
	return nil
}

// ptySizeQueue 将 <-chan TerminalSize 适配为 client-go 的 TerminalSizeQueue 接口。
// 当通道关闭后 Next 返回 nil，告诉 client-go 终止 SIGWINCH 下发。
type ptySizeQueue struct {
	ch <-chan TerminalSize
}

// Next 阻塞读取下一个 resize 事件；channel 关闭返回 nil。
func (q ptySizeQueue) Next() *remotecommand.TerminalSize {
	if q.ch == nil {
		// 一旦消费就阻塞到 ctx 取消——客户端未发 resize 时此 goroutine 静止。
		select {}
	}
	size, ok := <-q.ch
	if !ok {
		return nil
	}
	return &remotecommand.TerminalSize{Width: size.Width, Height: size.Height}
}

// GetPodLogs 获取 Pod 容器日志
func (k *k8sClient) GetPodLogs(ctx context.Context, namespace, podName, container string, tailLines int) (string, error) {
	opts := &corev1.PodLogOptions{
		Container: container,
	}
	if tailLines > 0 {
		lines := int64(tailLines)
		opts.TailLines = &lines
	}

	req := k.clientset.CoreV1().Pods(namespace).GetLogs(podName, opts)
	stream, err := req.Stream(ctx)
	if err != nil {
		return "", fmt.Errorf("获取日志流失败: %w", err)
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return "", fmt.Errorf("读取日志失败: %w", err)
	}
	return buf.String(), nil
}

// CaptureContainerRuntimeState 采集容器卷目录的归档内容，用于暂停后恢复实验容器状态。
func (k *k8sClient) CaptureContainerRuntimeState(ctx context.Context, namespace, podName, container string, mountPaths []string) (*RuntimeContainerState, error) {
	state := &RuntimeContainerState{
		ContainerName: container,
		PodName:       podName,
		Status:        2,
	}
	for _, mountPath := range mountPaths {
		normalizedPath := strings.TrimSpace(mountPath)
		if normalizedPath == "" || normalizedPath == "/" {
			continue
		}
		relativePath := strings.TrimPrefix(path.Clean(normalizedPath), "/")
		if relativePath == "" {
			continue
		}
		command := fmt.Sprintf("set -e; if [ -e %s ]; then tar -C / -czf - -- %s; fi", shellQuote(path.Clean(normalizedPath)), shellQuote(relativePath))
		result, err := k.execInPod(ctx, namespace, podName, container, []string{"/bin/sh", "-c", command}, nil)
		if err != nil {
			return nil, fmt.Errorf("采集容器目录 %s 失败: %w", normalizedPath, err)
		}
		if result.ExitCode != 0 {
			return nil, fmt.Errorf("采集容器目录 %s 失败: %s", normalizedPath, strings.TrimSpace(result.Stderr))
		}
		state.VolumeArchives = append(state.VolumeArchives, RuntimeVolumeArchive{
			MountPath:   path.Clean(normalizedPath),
			ArchiveData: base64.StdEncoding.EncodeToString([]byte(result.Stdout)),
		})
	}
	return state, nil
}

// RestoreContainerRuntimeState 将快照中的卷归档内容恢复到新建容器。
func (k *k8sClient) RestoreContainerRuntimeState(ctx context.Context, namespace, podName, container string, state *RuntimeContainerState) error {
	if state == nil {
		return nil
	}
	for _, archive := range state.VolumeArchives {
		if strings.TrimSpace(archive.ArchiveData) == "" || strings.TrimSpace(archive.MountPath) == "" {
			continue
		}
		decoded, err := base64.StdEncoding.DecodeString(archive.ArchiveData)
		if err != nil {
			return fmt.Errorf("解析容器目录归档失败: %w", err)
		}
		parentDir := path.Dir(path.Clean(archive.MountPath))
		command := fmt.Sprintf("set -e; mkdir -p %s; tar -xzf - -C /", shellQuote(parentDir))
		result, err := k.execInPod(ctx, namespace, podName, container, []string{"/bin/sh", "-c", command}, bytes.NewReader(decoded))
		if err != nil {
			return fmt.Errorf("恢复容器目录 %s 失败: %w", archive.MountPath, err)
		}
		if result.ExitCode != 0 {
			return fmt.Errorf("恢复容器目录 %s 失败: %s", archive.MountPath, strings.TrimSpace(result.Stderr))
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 资源监控
// ---------------------------------------------------------------------------

// GetResourceUsage 获取命名空间资源使用情况
func (k *k8sClient) GetResourceUsage(ctx context.Context, namespace string) (*ResourceUsage, error) {
	podList, err := k.clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	usedCPU, usedMem, usedStorage, podCount, _, _ := summarizePodResources(podList.Items)
	totalCPU := ""
	totalMem := ""
	totalStorage := ""
	if quotaList, quotaErr := k.clientset.CoreV1().ResourceQuotas(namespace).List(ctx, metav1.ListOptions{}); quotaErr == nil {
		for _, quota := range quotaList.Items {
			if totalCPU == "" {
				totalCPU = readQuotaQuantity(quota.Status.Hard, corev1.ResourceLimitsCPU)
			}
			if totalMem == "" {
				totalMem = readQuotaQuantity(quota.Status.Hard, corev1.ResourceLimitsMemory)
			}
			if totalStorage == "" {
				totalStorage = readQuotaQuantity(quota.Status.Hard, corev1.ResourceLimitsEphemeralStorage)
			}
		}
	}

	if totalCPU == "" {
		totalCPU = "0"
	}
	if totalMem == "" {
		totalMem = "0"
	}
	if totalStorage == "" {
		totalStorage = "0"
	}

	return &ResourceUsage{
		CPUUsed:   formatMilliCPU(usedCPU),
		CPUTotal:  totalCPU,
		MemUsed:   formatBytes(usedMem),
		MemTotal:  totalMem,
		DiskUsed:  formatBytes(usedStorage),
		DiskTotal: totalStorage,
		PodCount:  podCount,
	}, nil
}

// GetNodeStatus 获取所有节点状态
func (k *k8sClient) GetNodeStatus(ctx context.Context) ([]*NodeStatus, error) {
	nodeList, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	podList, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	podsByNode := make(map[string][]corev1.Pod)
	for _, pod := range podList.Items {
		podsByNode[pod.Spec.NodeName] = append(podsByNode[pod.Spec.NodeName], pod)
	}

	result := make([]*NodeStatus, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		status := "NotReady"
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				status = "Ready"
				break
			}
		}

		nodePods := podsByNode[node.Name]
		usedCPU, usedMem, _, podCount, containerCount, runningContainers := summarizePodResources(nodePods)
		podCapacity := 0
		if quantity, ok := node.Status.Capacity[corev1.ResourcePods]; ok {
			podCapacity = int(quantity.Value())
		}

		result = append(result, &NodeStatus{
			Name:              node.Name,
			Status:            status,
			KubeletVersion:    node.Status.NodeInfo.KubeletVersion,
			CPUUsed:           formatMilliCPU(usedCPU),
			CPUTotal:          node.Status.Capacity.Cpu().String(),
			CPUAllocatable:    node.Status.Allocatable.Cpu().String(),
			MemUsed:           formatBytes(usedMem),
			MemTotal:          node.Status.Capacity.Memory().String(),
			MemAllocatable:    node.Status.Allocatable.Memory().String(),
			DiskUsed:          "0",
			DiskTotal:         "0",
			PodCount:          podCount,
			ContainerCount:    containerCount,
			RunningContainers: runningContainers,
			PodCapacity:       podCapacity,
		})
	}
	return result, nil
}

// GetClusterStatus 获取集群整体状态
func (k *k8sClient) GetClusterStatus(ctx context.Context) (*ClusterStatus, error) {
	nodeList, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	podList, err := k.clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	namespaceList, err := k.clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	totalNodes := len(nodeList.Items)
	readyNodes := 0
	var totalCPU, totalMem int64
	for _, node := range nodeList.Items {
		for _, cond := range node.Status.Conditions {
			if cond.Type == corev1.NodeReady && cond.Status == corev1.ConditionTrue {
				readyNodes++
				break
			}
		}
		if cpu := node.Status.Capacity.Cpu(); cpu != nil {
			totalCPU += cpu.MilliValue()
		}
		if mem := node.Status.Capacity.Memory(); mem != nil {
			totalMem += mem.Value()
		}
	}

	usedCPU, usedMem, _, _, _, _ := summarizePodResources(podList.Items)
	runningPods := 0
	pendingPods := 0
	failedPods := 0
	for _, pod := range podList.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			runningPods++
		case corev1.PodPending:
			pendingPods++
		case corev1.PodFailed:
			failedPods++
		}
	}

	return &ClusterStatus{
		TotalNodes:   totalNodes,
		ReadyNodes:   readyNodes,
		TotalCPU:     formatMilliCPU(totalCPU),
		UsedCPU:      formatMilliCPU(usedCPU),
		TotalMemory:  formatBytes(totalMem),
		UsedMemory:   formatBytes(usedMem),
		TotalStorage: "0",
		UsedStorage:  "0",
		TotalPods:    len(podList.Items),
		RunningPods:  runningPods,
		PendingPods:  pendingPods,
		FailedPods:   failedPods,
		Namespaces:   len(namespaceList.Items),
	}, nil
}

// ---------------------------------------------------------------------------
// 镜像管理
// ---------------------------------------------------------------------------

// PrePullImage 在指定节点上预拉取镜像
// 通过创建 DaemonSet 或短生命周期 Pod 实现
func (k *k8sClient) PrePullImage(ctx context.Context, imageURL string, nodes []string) error {
	targetNodes, err := k.resolveReadyPrePullNodes(ctx, nodes)
	if err != nil {
		return err
	}
	for _, nodeName := range targetNodes {
		podName := fmt.Sprintf("prepull-%s-%d", sanitizeName(nodeName), time.Now().Unix())
		pod := &corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Name:      podName,
				Namespace: "default",
				Labels: map[string]string{
					"app":   "lenschain-prepull",
					"image": sanitizeName(imageURL),
					"node":  sanitizeName(nodeName),
				},
			},
			Spec: corev1.PodSpec{
				NodeName:      nodeName,
				RestartPolicy: corev1.RestartPolicyNever,
				Containers: []corev1.Container{
					{
						Name:    "pull",
						Image:   imageURL,
						Command: []string{"/bin/sh", "-c", "echo pulled && exit 0"},
						Resources: corev1.ResourceRequirements{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU:    resource.MustParse("10m"),
								corev1.ResourceMemory: resource.MustParse("16Mi"),
							},
						},
					},
				},
			},
		}
		_, err = k.clientset.CoreV1().Pods("default").Create(ctx, pod, metav1.CreateOptions{})
		if err != nil {
			return fmt.Errorf("节点 %s 预拉取失败: %w", nodeName, err)
		}
	}
	return nil
}

// GetImagePullStatus 查询镜像在各节点的拉取状态
func (k *k8sClient) GetImagePullStatus(ctx context.Context, imageURL string) ([]*ImagePullNodeStatus, error) {
	nodeList, err := k.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	prePullPods, err := k.clientset.CoreV1().Pods("default").List(ctx, metav1.ListOptions{
		LabelSelector: "app=lenschain-prepull,image=" + sanitizeName(imageURL),
	})
	if err != nil {
		return nil, err
	}

	result := make([]*ImagePullNodeStatus, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		status := "not_pulled"
		var pulledAt *time.Time
		var cacheSizeBytes int64
		for _, img := range node.Status.Images {
			cacheSizeBytes += img.SizeBytes
			for _, name := range img.Names {
				if name == imageURL || strings.HasPrefix(name, imageURL+":") || strings.HasSuffix(name, "/"+imageURL) {
					status = "pulled"
					now := time.Now().UTC()
					pulledAt = &now
					break
				}
			}
			if status == "pulled" {
				break
			}
		}
		progress := ""
		errMessage := ""
		if status != "pulled" {
			status, progress, errMessage = derivePrePullPodStatus(node.Name, prePullPods.Items)
		}
		result = append(result, &ImagePullNodeStatus{
			NodeName:      node.Name,
			Status:        status,
			Progress:      progress,
			Error:         errMessage,
			PulledAt:      pulledAt,
			NodeCacheSize: formatBytes(cacheSizeBytes),
		})
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 内部辅助
// ---------------------------------------------------------------------------

// podToPodStatus 将 K8s Pod 对象转换为 PodStatus
func (k *k8sClient) podToPodStatus(pod *corev1.Pod) *PodStatus {
	ps := &PodStatus{
		PodName:    pod.Name,
		Namespace:  pod.Namespace,
		NodeName:   pod.Spec.NodeName,
		Status:     string(pod.Status.Phase),
		Reason:     pod.Status.Reason,
		Message:    pod.Status.Message,
		InternalIP: pod.Status.PodIP,
	}
	if pod.Status.StartTime != nil {
		ps.StartedAt = pod.Status.StartTime.Format(time.RFC3339)
	}
	for _, status := range pod.Status.ContainerStatuses {
		if status.State.Waiting != nil {
			ps.Reason = status.State.Waiting.Reason
			ps.Message = status.State.Waiting.Message
			break
		}
		if status.State.Terminated != nil {
			ps.Reason = status.State.Terminated.Reason
			ps.Message = status.State.Terminated.Message
			if ps.Message == "" {
				ps.Message = status.State.Terminated.String()
			}
			break
		}
		if status.LastTerminationState.Terminated != nil {
			ps.Reason = status.LastTerminationState.Terminated.Reason
			ps.Message = status.LastTerminationState.Terminated.Message
			if ps.Message == "" {
				ps.Message = status.LastTerminationState.Terminated.String()
			}
			break
		}
	}

	// 汇总容器资源限制
	var cpuTotal, memTotal int64
	for _, c := range pod.Spec.Containers {
		if cpu := c.Resources.Limits.Cpu(); cpu != nil {
			cpuTotal += cpu.MilliValue()
		}
		if mem := c.Resources.Limits.Memory(); mem != nil {
			memTotal += mem.Value()
		}
	}
	ps.CPUUsage = fmt.Sprintf("%dm", cpuTotal)
	ps.MemUsage = formatBytes(memTotal)

	return ps
}

// buildLabelSelector 将 map 转换为 K8s 标签选择器字符串
func buildLabelSelector(labels map[string]string) string {
	parts := make([]string, 0, len(labels))
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(parts, ",")
}

// derivePrePullPodStatus 根据预拉取任务 Pod 推导节点侧拉取状态。
func derivePrePullPodStatus(nodeName string, pods []corev1.Pod) (string, string, string) {
	var latest *corev1.Pod
	for index := range pods {
		pod := &pods[index]
		if pod.Spec.NodeName != nodeName {
			continue
		}
		if latest == nil || pod.CreationTimestamp.After(latest.CreationTimestamp.Time) {
			latest = pod
		}
	}
	if latest == nil {
		return "not_pulled", "", ""
	}

	switch latest.Status.Phase {
	case corev1.PodPending:
		return "pulling", "preparing", ""
	case corev1.PodRunning:
		return "pulling", "running", ""
	case corev1.PodSucceeded:
		completedAt := latest.Status.ContainerStatuses
		if len(completedAt) > 0 && completedAt[0].State.Terminated != nil && completedAt[0].State.Terminated.ExitCode == 0 {
			return "pulling", "verifying", ""
		}
		return "pulling", "verifying", ""
	case corev1.PodFailed:
		message := latest.Status.Message
		if len(latest.Status.ContainerStatuses) > 0 && latest.Status.ContainerStatuses[0].State.Terminated != nil && latest.Status.ContainerStatuses[0].State.Terminated.Message != "" {
			message = latest.Status.ContainerStatuses[0].State.Terminated.Message
		}
		return "failed", "", message
	default:
		return "not_pulled", "", ""
	}
}

// execInPod 执行容器命令，并按需向 stdin 注入内容。
func (k *k8sClient) execInPod(ctx context.Context, namespace, podName, container string, command []string, stdin io.Reader) (*ExecResult, error) {
	req := k.clientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		VersionedParams(&corev1.PodExecOptions{
			Container: container,
			Command:   command,
			Stdin:     stdin != nil,
			Stdout:    true,
			Stderr:    true,
			TTY:       false,
		}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(k.restConfig, "POST", req.URL())
	if err != nil {
		return nil, fmt.Errorf("创建远程执行器失败: %w", err)
	}

	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:  stdin,
		Stdout: &stdout,
		Stderr: &stderr,
	})

	exitCode := 0
	if err != nil {
		exitCode = 1
	}
	return &ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}, nil
}

// sanitizeName 将字符串转换为合法的 K8s 名称片段
func sanitizeName(s string) string {
	r := strings.NewReplacer("/", "-", ":", "-", ".", "-", "_", "-")
	name := r.Replace(strings.ToLower(s))
	if len(name) > 50 {
		name = name[:50]
	}
	return strings.Trim(name, "-")
}

// shellQuote 将路径安全包裹为 POSIX shell 单引号字符串。
func shellQuote(raw string) string {
	return "'" + strings.ReplaceAll(raw, "'", "'\"'\"'") + "'"
}

// resolveReadyPrePullNodes 返回需要参与预拉取的 Ready 节点列表。
// 当未显式指定节点时，会自动选择当前所有 Ready 节点。
func (k *k8sClient) resolveReadyPrePullNodes(ctx context.Context, requested []string) ([]string, error) {
	nodes, err := k.GetNodeStatus(ctx)
	if err != nil {
		return nil, err
	}
	readySet := make(map[string]struct{}, len(nodes))
	for _, node := range nodes {
		if node == nil || node.Status != "Ready" {
			continue
		}
		readySet[node.Name] = struct{}{}
	}
	if len(requested) == 0 {
		result := make([]string, 0, len(readySet))
		for nodeName := range readySet {
			result = append(result, nodeName)
		}
		sort.Strings(result)
		return result, nil
	}

	result := make([]string, 0, len(requested))
	for _, nodeName := range requested {
		if _, ok := readySet[nodeName]; ok {
			result = append(result, nodeName)
		}
	}
	sort.Strings(result)
	return result, nil
}

// boolPtr 返回布尔指针，便于填充 K8s 安全上下文字段。
func boolPtr(value bool) *bool {
	return &value
}

// waitForTCPInitImage 是依赖等待 initContainer 使用的镜像。busybox 自带 nc(netcat)，
// 体积仅几 MB；imagePullPolicy=IfNotPresent 让首个 Pod 拉一次后整个集群复用，避免
// 私有 registry 不可达时无法启动业务实验。
const waitForTCPInitImage = "busybox:1.36"

// buildWaitForTCPInitContainers 把 ContainerSpec.WaitForTCP 翻译为单个 Pod 级
// initContainer，串行 nc -z 等待所有依赖 host:port 可建联后退出。
//
// 设计取舍：
//   - 单 init 容器、串行等待——比"每个依赖一个 init"语义等价但更省调度开销；
//     init 串行执行本就是 K8s 默认行为，单容器内 for 循环更直接。
//   - 失败策略：nc 不可用 / DNS 解析失败 / 端口永不就绪 → init 阻塞 → Pod 长时间
//     处于 PodInitializing → DeployPod 等待超时返回错误，错误会冒泡到 instance_service
//     的 K8s 部署失败处理，与现有错误链路对齐，无需新增分支。
//   - 不与 readinessProbe 重复：readiness 是"运行中容器是否准备好接流量"，覆盖不了
//     "依赖未就绪导致主进程一启动就退出"这一启动时窗口。两者职责不重合。
//
// 返回 nil 表示无依赖，调用方直接传给 PodSpec.InitContainers 也安全（nil slice）。
func buildWaitForTCPInitContainers(specs []ContainerSpec) []corev1.Container {
	targets := make([]string, 0)
	seen := make(map[string]bool)
	for _, cs := range specs {
		for _, t := range cs.WaitForTCP {
			t = strings.TrimSpace(t)
			if t == "" || seen[t] {
				continue
			}
			seen[t] = true
			targets = append(targets, t)
		}
	}
	if len(targets) == 0 {
		return nil
	}
	// 用引号 + 空格分隔的 shell 列表；逐个 nc -z 等待，间隔 1s 重试。
	// `nc -w 2` 设单次 connect 超时 2s，避免 DNS 解析慢时白等。
	listLiteral := "'" + strings.Join(targets, "' '") + "'"
	script := fmt.Sprintf(
		"set -e; for tgt in %s; do host=${tgt%%:*}; port=${tgt##*:}; "+
			"echo \"[wait-for-tcp] $tgt\"; "+
			"until nc -z -w 2 \"$host\" \"$port\" 2>/dev/null; do sleep 1; done; "+
			"echo \"[wait-for-tcp] $tgt ready\"; done",
		listLiteral,
	)
	return []corev1.Container{
		{
			Name:            "wait-for-deps",
			Image:           waitForTCPInitImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"/bin/sh", "-c", script},
			Resources: corev1.ResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("100m"),
					corev1.ResourceMemory: resource.MustParse("32Mi"),
				},
				Requests: corev1.ResourceList{
					corev1.ResourceCPU:    resource.MustParse("50m"),
					corev1.ResourceMemory: resource.MustParse("16Mi"),
				},
			},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: boolPtr(false),
				Privileged:               boolPtr(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
			},
		},
	}
}

// protocolPtr 返回协议指针，便于拼装 NetworkPolicy 端口。
func protocolPtr(protocol corev1.Protocol) *corev1.Protocol {
	return &protocol
}

// appendResourceQuantity 将合法的资源量写入 K8s 资源列表。
func appendResourceQuantity(resources corev1.ResourceList, name corev1.ResourceName, quantity string) {
	if strings.TrimSpace(quantity) == "" || quantity == "0" {
		return
	}
	resources[name] = resource.MustParse(quantity)
}

// summarizePodResources 汇总一组 Pod 的资源限制占用、Pod 数量和容器数量。
func summarizePodResources(pods []corev1.Pod) (cpuMilli int64, memoryBytes int64, storageBytes int64, podCount int, containerCount int, runningContainers int) {
	for _, pod := range pods {
		podCount++
		containerCount += len(pod.Spec.Containers)
		for _, status := range pod.Status.ContainerStatuses {
			if status.State.Running != nil {
				runningContainers++
			}
		}
		for _, container := range pod.Spec.Containers {
			if cpu := container.Resources.Limits.Cpu(); cpu != nil {
				cpuMilli += cpu.MilliValue()
			}
			if memory := container.Resources.Limits.Memory(); memory != nil {
				memoryBytes += memory.Value()
			}
			if storage := container.Resources.Limits.StorageEphemeral(); storage != nil {
				storageBytes += storage.Value()
			}
		}
	}
	return cpuMilli, memoryBytes, storageBytes, podCount, containerCount, runningContainers
}

// readQuotaQuantity 读取 ResourceQuota 中指定资源名的硬限制。
func readQuotaQuantity(resources corev1.ResourceList, name corev1.ResourceName) string {
	quantity, ok := resources[name]
	if !ok {
		return ""
	}
	return quantity.String()
}

// formatMilliCPU 将毫核数格式化为 CPU 字符串。
func formatMilliCPU(milli int64) string {
	return fmt.Sprintf("%dm", milli)
}

// formatBytes 将字节数格式化为可读字符串
func formatBytes(b int64) string {
	const (
		kb = 1024
		mb = kb * 1024
		gb = mb * 1024
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%dGi", b/gb)
	case b >= mb:
		return fmt.Sprintf("%dMi", b/mb)
	case b >= kb:
		return fmt.Sprintf("%dKi", b/kb)
	default:
		return fmt.Sprintf("%d", b)
	}
}
