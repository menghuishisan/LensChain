// k8s_service.go
// 模块04 — 实验环境：K8s 编排适配契约
// 定义模块04调用 Kubernetes 所需的最小接口和数据结构，供 service 层做实例生命周期与监控编排
// 该文件只声明外部系统适配契约，不承载实验业务规则，也不替代 internal/pkg 基础能力

package experiment

import (
	"context"
	"time"
)

// K8sService K8s 编排服务接口
// 封装所有 K8s 操作，service 层通过此接口与 K8s 交互
type K8sService interface {
	// 命名空间管理
	CreateNamespace(ctx context.Context, name string, labels map[string]string, resourceSpec *NamespaceResourceSpec) error
	DeleteNamespace(ctx context.Context, name string) error

	// Pod 管理
	DeployPod(ctx context.Context, req *DeployPodRequest) (*DeployPodResponse, error)
	DeletePod(ctx context.Context, namespace, podName string) error
	GetPodStatus(ctx context.Context, namespace, podName string) (*PodStatus, error)
	ListPods(ctx context.Context, namespace string, labels map[string]string) ([]*PodStatus, error)

	// 容器操作
	ExecInPod(ctx context.Context, namespace, podName, container, command string) (*ExecResult, error)
	GetPodLogs(ctx context.Context, namespace, podName, container string, tailLines int) (string, error)
	CaptureContainerRuntimeState(ctx context.Context, namespace, podName, container string, mountPaths []string) (*RuntimeContainerState, error)
	RestoreContainerRuntimeState(ctx context.Context, namespace, podName, container string, state *RuntimeContainerState) error

	// 资源监控
	GetResourceUsage(ctx context.Context, namespace string) (*ResourceUsage, error)
	GetNodeStatus(ctx context.Context) ([]*NodeStatus, error)
	GetClusterStatus(ctx context.Context) (*ClusterStatus, error)

	// 镜像管理
	PrePullImage(ctx context.Context, imageURL string, nodes []string) error
	GetImagePullStatus(ctx context.Context, imageURL string) ([]*ImagePullNodeStatus, error)
}

// DeployPodRequest Pod 部署请求
type DeployPodRequest struct {
	Namespace     string
	PodName       string
	Containers    []ContainerSpec
	Labels        map[string]string
	NetworkPolicy *NetworkPolicySpec
	Collector     *CollectorSidecarSpec
}

// ContainerSpec 容器规格
type ContainerSpec struct {
	Name        string
	Image       string
	Ports       []PortSpec
	EnvVars     map[string]string
	Volumes     []VolumeSpec
	CPULimit    string
	MemoryLimit string
	Command     []string
	Args        []string
}

// PortSpec 端口规格
type PortSpec struct {
	ContainerPort int
	Protocol      string
	ServicePort   int
}

// VolumeSpec 卷规格
type VolumeSpec struct {
	Name      string
	MountPath string
	SubPath   string
	ReadOnly  bool
}

// NetworkPolicySpec 网络策略规格
type NetworkPolicySpec struct {
	AllowIngress                 []string
	AllowEgress                  []string
	AllowSameNamespace           bool
	AllowNamespaceLabelSelectors []map[string]string
}

// CollectorSidecarSpec 定义混合实验自动注入的 Collector Agent sidecar。
// 该结构同时携带采集目标、Collector 生态编码和连接 SimEngine Core 所需的会话信息。
type CollectorSidecarSpec struct {
	TargetContainer  string
	Ecosystem        string
	DataSourceConfig []byte
	SessionID        string
	CoreWebSocketURL string
}

// DeployPodResponse Pod 部署响应
type DeployPodResponse struct {
	PodName    string
	Namespace  string
	InternalIP string
	Status     string
}

// PodStatus Pod 状态
type PodStatus struct {
	PodName    string
	Namespace  string
	NodeName   string
	Status     string
	Reason     string
	Message    string
	InternalIP string
	CPUUsage   string
	MemUsage   string
	StartedAt  string
}

// ExecResult 命令执行结果
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// RuntimeContainerState 表示实验容器的可恢复运行态快照。
// 当前聚焦卷目录文件状态，用于暂停后恢复实验容器状态。
type RuntimeContainerState struct {
	ContainerName  string                 `json:"container_name"`
	PodName        string                 `json:"pod_name"`
	InternalIP     string                 `json:"internal_ip,omitempty"`
	Status         int                    `json:"status"`
	VolumeArchives []RuntimeVolumeArchive `json:"volume_archives,omitempty"`
}

// RuntimeVolumeArchive 表示单个挂载目录的归档内容。
// ArchiveData 为 tar.gz 的 base64 编码内容，仅写入对象存储归档。
type RuntimeVolumeArchive struct {
	MountPath   string `json:"mount_path"`
	ArchiveData string `json:"archive_data,omitempty"`
}

// ResourceUsage 资源使用情况
type ResourceUsage struct {
	CPUUsed   string
	CPUTotal  string
	MemUsed   string
	MemTotal  string
	DiskUsed  string
	DiskTotal string
	PodCount  int
}

// NodeStatus 节点状态
type NodeStatus struct {
	Name              string
	Status            string
	KubeletVersion    string
	CPUUsed           string
	CPUTotal          string
	CPUAllocatable    string
	MemUsed           string
	MemTotal          string
	MemAllocatable    string
	DiskUsed          string
	DiskTotal         string
	PodCount          int
	ContainerCount    int
	RunningContainers int
	PodCapacity       int
}

// ClusterStatus 集群状态
type ClusterStatus struct {
	TotalNodes   int
	ReadyNodes   int
	TotalCPU     string
	UsedCPU      string
	TotalMemory  string
	UsedMemory   string
	TotalStorage string
	UsedStorage  string
	TotalPods    int
	RunningPods  int
	PendingPods  int
	FailedPods   int
	Namespaces   int
}

// ImagePullNodeStatus 镜像拉取节点状态
type ImagePullNodeStatus struct {
	NodeName      string
	Status        string
	Progress      string
	Error         string
	PulledAt      *time.Time
	NodeCacheSize string
}

// NamespaceResourceSpec 定义实验命名空间的资源隔离规格。
// 用于在创建命名空间时同步下发 ResourceQuota 和 LimitRange。
type NamespaceResourceSpec struct {
	HardCPU                 string
	HardMemory              string
	HardStorage             string
	DefaultContainerCPU     string
	DefaultContainerMemory  string
	DefaultContainerStorage string
}

// 真实实现见 k8s_client.go（基于 client-go）
