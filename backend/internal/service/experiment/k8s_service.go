// k8s_service.go
// 模块04 — 实验环境：K8s 编排适配契约
// 定义模块04调用 Kubernetes 所需的最小接口和数据结构，供 service 层做实例生命周期与监控编排
// 该文件只声明外部系统适配契约，不承载实验业务规则，也不替代 internal/pkg 基础能力

package experiment

import (
	"context"
	"io"
	"net"
	"time"
)

// K8sService K8s 编排服务接口
// 封装所有 K8s 操作，service 层通过此接口与 K8s 交互
type K8sService interface {
	// 命名空间管理
	CreateNamespace(ctx context.Context, name string, labels map[string]string, resourceSpec *NamespaceResourceSpec) error
	DeleteNamespace(ctx context.Context, name string) error
	// DeletePodsInNamespace 仅删 Pod，保留 Service / NetworkPolicy / PVC，详见实现注释。
	// 暂停实验专用入口；Destroy 仍走 DeleteNamespace。
	DeletePodsInNamespace(ctx context.Context, namespace string) error

	// Pod 管理
	DeployPod(ctx context.Context, req *DeployPodRequest) (*DeployPodResponse, error)
	DeletePod(ctx context.Context, namespace, podName string) error
	GetPodStatus(ctx context.Context, namespace, podName string) (*PodStatus, error)
	ListPods(ctx context.Context, namespace string, labels map[string]string) ([]*PodStatus, error)

	// 容器操作
	ExecInPod(ctx context.Context, namespace, podName, container, command string) (*ExecResult, error)
	// ExecPodPTY 在 Pod 容器内启动一个 PTY 进程（默认 /bin/sh），通过 K8s exec subresource
	// 的 SPDY 流双向桥接 stdin/stdout 与终端尺寸变更。用于 Web 终端：浏览器 ↔ 后端 WS ↔
	// K8s exec ↔ 目标容器 PTY，工具按容器自然就绪（redis-cli 在 redis 容器、psql 在
	// postgres 容器、geth attach 在 geth 容器）。
	//
	// 实现合约：
	//   - command 为空时使用 ["/bin/sh"]；如目标容器有 /bin/bash 调用方可显式传入。
	//   - stdin/stdout 必须由调用方提供 io.Reader/io.Writer；stderr 合并到 stdout（PTY 模式
	//     下与终端语义一致，不需要拆分）。
	//   - resize 通过 TerminalSize 通道异步下发到 K8s，channel 关闭表示终端关闭。
	//   - 方法在 PTY 进程退出或 ctx 取消时返回；不在内部启动 goroutine，调用方负责并发。
	ExecPodPTY(ctx context.Context, req *ExecPodPTYRequest) error
	GetPodLogs(ctx context.Context, namespace, podName, container string, tailLines int) (string, error)
	CaptureContainerRuntimeState(ctx context.Context, namespace, podName, container string, mountPaths []string) (*RuntimeContainerState, error)
	RestoreContainerRuntimeState(ctx context.Context, namespace, podName, container string, state *RuntimeContainerState) error

	// DialPodPort 通过 K8s API 的 SPDY portforward 隧道连接到指定 Pod 端口，返回 net.Conn。
	// 用于 backend 代理 HTTP / iframe 反代到工具镜像（code-server / blockscout / VNC 等）；
	// 终端 PTY 不再走此路径——见 ExecPodPTY。
	// 设计依据见 docs/modules/09-部署与运维/02-基础设施设计.md §2.4，本仓库不允许直拨 Pod IP /
	// Service ClusterIP 或暴露 NodePort 替代该路径。
	DialPodPort(ctx context.Context, namespace, podName string, port int) (net.Conn, error)

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
	// WaitForTCP 在主容器启动前等待这些 host:port 都可建立 TCP 连接。
	// 由 K8sService 实现端翻译为一个 busybox initContainer 跑 `nc -z` 循环。
	// 取代"启动顺序+sleep"的脆弱协调：postgres Pod 已 Running 不等于 postgres 已就绪，
	// CouchDB / fabric-orderer 等都有显著的应用层冷启动时间。
	WaitForTCP []string
	// IsInitContainer 为 TRUE 时该容器进入 Pod 的 InitContainers 而非主 Containers。
	// 用于业务自定义的 bootstrap 容器（例如 cryptogen 生成 Fabric MSP），与本结构里
	// 由 WaitForTCP 自动生成的 wait-for-tcp init 容器并列。详见 docs/modules/04-实验环境/
	// 02-数据库设计.md §2.5 Pod 打包与卷共享语义。
	IsInitContainer bool
}

// PortSpec 端口规格
type PortSpec struct {
	ContainerPort int
	Protocol      string
	ServicePort   int
}

// VolumeSpec 卷规格。
//
// JSON 标签使用 snake_case 以便 template_containers.volumes (JSONB) 直接反序列化；
// 与 serviceDiscoveryPort / DTO 等本模块其余结构保持一致的 JSON 约定。
//
// 卷类型由 Size 字段决定（与文档 §6.3 "按镜像 default_volumes 自动创建 PV" 对齐）：
//   - Size 非空（如 "5Gi"）：标识来自 image.default_volumes 的持久数据卷，运行时
//     创建独立 PVC，Pod 引用之；Pod 重启 / 节点漂移后数据保留，由 namespace 级联
//     删除统一清理。典型场景：code-server 工作区、链节点数据、DB 数据。
//   - Size 空：标识 template_container 显式声明的 Pod 内共享卷，运行时使用 emptyDir，
//     Pod 销毁即回收。典型场景：Fabric initContainer cryptogen 产物 → peer 主容器读取。
//
// 这一区分让 manifest 的"持久 vs 共享"语义直接落到 K8s 资源类型，无需额外 schema 字段。
type VolumeSpec struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	SubPath   string `json:"sub_path"`
	ReadOnly  bool   `json:"read_only"`
	Size      string `json:"size,omitempty"`
}

// NetworkPolicySpec 网络策略规格
type NetworkPolicySpec struct {
	AllowIngress                 []string
	AllowEgress                  []string
	AllowDNS                     bool
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

// TerminalSize 终端窗口尺寸（与 client-go remotecommand.TerminalSize 对齐）。
type TerminalSize struct {
	Width  uint16
	Height uint16
}

// ExecPodPTYRequest 描述一次交互式 PTY exec 调用。
//
// Command 留空时 K8sService 实现端会默认为 ["/bin/sh"]——这是绝大多数官方 / Alpine /
// distroless-debian 镜像内可用的最低公共集；若目标容器明确装有 bash 或自定义 shell，
// 调用方可以显式传入。
type ExecPodPTYRequest struct {
	Namespace string
	PodName   string
	Container string
	Command   []string
	Stdin     io.Reader
	Stdout    io.Writer
	// Resize 由调用方持有写端，K8sService 实现端只读消费。channel 关闭 = 终端关闭，
	// 实现端必须正确传递 SIGWINCH 而不在 channel 关闭时 panic。
	Resize <-chan TerminalSize
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
