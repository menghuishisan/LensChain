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
		return fmt.Errorf("%w: %v", errcode.ErrK8sNamespaceCreateFailed, err)
	}
	if err := k.applyNamespaceResourceIsolation(ctx, name, resourceSpec); err != nil {
		return err
	}
	return nil
}

// DeleteNamespace 删除 K8s 命名空间
func (k *k8sClient) DeleteNamespace(ctx context.Context, name string) error {
	return k.clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{})
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

// DeployPod 部署 Pod（含 Service 和可选 NetworkPolicy）
func (k *k8sClient) DeployPod(ctx context.Context, req *DeployPodRequest) (*DeployPodResponse, error) {
	// 构建容器列表
	containers := make([]corev1.Container, 0, len(req.Containers))
	for _, cs := range req.Containers {
		container := corev1.Container{
			Name:  cs.Name,
			Image: cs.Image,
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: boolPtr(false),
				Privileged:               boolPtr(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
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

		containers = append(containers, container)
	}

	if req.Collector != nil {
		containers = append(containers, k.buildCollectorContainer(req.Collector))
	}

	// 构建 Pod 卷（为每个 VolumeSpec 创建 emptyDir）
	volumes := make([]corev1.Volume, 0)
	volumeNameSet := make(map[string]bool)
	for _, cs := range req.Containers {
		for _, v := range cs.Volumes {
			if !volumeNameSet[v.Name] {
				volumes = append(volumes, corev1.Volume{
					Name: v.Name,
					VolumeSource: corev1.VolumeSource{
						EmptyDir: &corev1.EmptyDirVolumeSource{},
					},
				})
				volumeNameSet[v.Name] = true
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
			Containers:    containers,
			Volumes:       volumes,
			RestartPolicy: corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				SeccompProfile: &corev1.SeccompProfile{Type: corev1.SeccompProfileTypeRuntimeDefault},
			},
		},
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

	return &DeployPodResponse{
		PodName:    createdPod.Name,
		Namespace:  createdPod.Namespace,
		InternalIP: createdPod.Status.PodIP,
		Status:     string(createdPod.Status.Phase),
	}, nil
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
	if len(req.NetworkPolicy.AllowEgress) > 0 || req.NetworkPolicy.AllowSameNamespace || len(req.NetworkPolicy.AllowNamespaceLabelSelectors) > 0 {
		var egressRules []networkingv1.NetworkPolicyEgressRule
		namespacePeers := buildNamespacePeers(req.Namespace, req.NetworkPolicy)
		if len(namespacePeers) > 0 {
			egressRules = append(egressRules, networkingv1.NetworkPolicyEgressRule{
				To: namespacePeers,
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
		Name:  "collector-agent",
		Image: fmt.Sprintf(imageTemplate, spec.Ecosystem),
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
