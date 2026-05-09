// k8s_orchestrator.go
// scene.Orchestrator 接口的 K8s 生产实现。
// 职责（参见 docs/modules/04-实验环境/06.1-场景编排实施方案.md）：
//   - StartScene：按 sceneCode + 镜像 + 资源请求在 K8s 创建 Pod 与 Service
//   - 主动 gRPC HealthCheck 拨号确认场景就绪（不依赖 readinessProbe，避免镜像拉取后 grpc 未起的窗口期）
//   - 维护 (sessionID, sceneCode) -> *podConnection 连接池，会话内复用
//   - EvictScene / DestroySession：分别在场景重启与会话销毁时清理资源
//   - StartIdleReaper：周期回收孤儿 Pod（IdleTTL）
//   - Shutdown：进程退出时按 managed-by 标签兜底清理

package scene

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"

	simscenariov1 "github.com/lenschain/sim-engine/proto/gen/go/lenschain/sim_scenario/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	// scenarioContainerPort 场景算法容器内 gRPC 监听端口。
	// 与 deploy/docker/scenario.Dockerfile 的 EXPOSE 50100 一致。
	scenarioContainerPort int32 = 50100

	// scenarioContainerName 场景 Pod 内主容器名称。
	scenarioContainerName = "scenario"

	// labelManagedBy 标识由 sim-engine 创建的资源，便于审计和清理。
	labelManagedBy = "app.kubernetes.io/managed-by"
	labelComponent = "app.kubernetes.io/component"
	labelSession   = "lenschain.io/session"
	labelSceneCode = "lenschain.io/scene-code"

	// healthCheckRetryInterval 主动 gRPC HealthCheck 探测间隔。
	healthCheckRetryInterval = 500 * time.Millisecond

	// reaperInterval 空闲回收循环的扫描间隔。最低粒度 1 分钟，避免过度调用 K8s API。
	reaperInterval = 1 * time.Minute
)

// OrchestratorConfig 描述 K8s 编排所需的全部参数。
// 来自 internal/config 包，由 main.go 加载 yaml 后注入。
type OrchestratorConfig struct {
	// InCluster 为 true 时使用集群内 ServiceAccount 配置，false 走 KubeconfigPath。
	InCluster bool
	// KubeconfigPath 集群外 kubeconfig 文件路径；空走 ~/.kube/config。
	KubeconfigPath string
	// Namespace 场景 Pod / Service 所在命名空间（与 SimEngine Core 同 ns）。
	Namespace string
	// PullSecretName 镜像拉取凭据 Secret 名称，复用平台级 imagePullSecrets。
	PullSecretName string
	// ReadyTimeout Pod 启动 + gRPC HealthCheck 通过的总超时（对齐文档 §6.4 = 10s）。
	ReadyTimeout time.Duration
	// IdleTTL 场景 Pod 空闲多久后被自动回收（仅当会话已销毁但 Pod 残留）。
	IdleTTL time.Duration
	// DefaultCPU / DefaultMemory 当 SceneConfig 未提供资源请求时的默认值。
	DefaultCPU    string
	DefaultMemory string
	// LimitCPU / LimitMemory 资源上限。
	LimitCPU    string
	LimitMemory string
}

// podConnection 表示一个会话内已建立的场景 Pod 与 gRPC 连接。
type podConnection struct {
	podName     string
	serviceName string
	conn        *grpc.ClientConn
}

// K8sOrchestrator 通过 K8s API 按需启停场景算法容器。
// 实现 scene.Orchestrator 接口。
type K8sOrchestrator struct {
	clientset *kubernetes.Clientset
	cfg       OrchestratorConfig

	mu   sync.Mutex
	pool map[string]*podConnection // key: sessionID + "::" + sceneCode
}

// NewK8sOrchestrator 创建编排器。
// in_cluster=false 且 kubeconfig 为空时尝试 ~/.kube/config。
func NewK8sOrchestrator(cfg OrchestratorConfig) (*K8sOrchestrator, error) {
	if strings.TrimSpace(cfg.Namespace) == "" {
		return nil, errors.New("orchestrator.namespace 不能为空")
	}
	if cfg.ReadyTimeout <= 0 {
		cfg.ReadyTimeout = 10 * time.Second
	}
	if cfg.IdleTTL <= 0 {
		cfg.IdleTTL = 10 * time.Minute
	}
	if cfg.DefaultCPU == "" {
		cfg.DefaultCPU = "100m"
	}
	if cfg.DefaultMemory == "" {
		cfg.DefaultMemory = "128Mi"
	}
	if cfg.LimitCPU == "" {
		cfg.LimitCPU = "500m"
	}
	if cfg.LimitMemory == "" {
		cfg.LimitMemory = "256Mi"
	}

	restCfg, err := buildRESTConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("构建 K8s 客户端配置失败: %w", err)
	}
	clientset, err := kubernetes.NewForConfig(restCfg)
	if err != nil {
		return nil, fmt.Errorf("创建 K8s clientset 失败: %w", err)
	}
	return &K8sOrchestrator{
		clientset: clientset,
		cfg:       cfg,
		pool:      make(map[string]*podConnection),
	}, nil
}

// StartScene 是 scene.Orchestrator 接口实现。按需启动场景 Pod 并返回 gRPC 客户端：
//  1. 校验镜像 URL；
//  2. 检查连接池命中则直接复用；
//  3. 创建 Service（先于 Pod 以便后续 DNS 解析稳定），再创建 Pod；
//  4. 主动拨号 + HealthCheck 重试，直到通过或超时；
//  5. 入池并返回 ScenarioClient。
func (o *K8sOrchestrator) StartScene(ctx context.Context, config Config) (ScenarioClient, error) {
	if strings.TrimSpace(config.SessionID) == "" {
		return nil, errors.New("session_id 不能为空")
	}
	if strings.TrimSpace(config.SceneCode) == "" {
		return nil, errors.New("scene_code 不能为空")
	}
	if strings.TrimSpace(config.ContainerImageURL) == "" {
		return nil, fmt.Errorf("场景 %s 缺少容器镜像 URL", config.SceneCode)
	}

	key := poolKey(config.SessionID, config.SceneCode)

	o.mu.Lock()
	if pc, ok := o.pool[key]; ok {
		o.mu.Unlock()
		return o.wrapClient(pc), nil
	}
	o.mu.Unlock()

	resourceName := buildResourceName(config.SessionID, config.SceneCode)
	labels := map[string]string{
		labelManagedBy: "sim-engine",
		labelComponent: "scenario",
		labelSession:   sanitizeLabelValue(config.SessionID),
		labelSceneCode: sanitizeLabelValue(config.SceneCode),
	}

	startCtx, cancel := context.WithTimeout(ctx, o.cfg.ReadyTimeout)
	defer cancel()

	clusterIP, err := o.ensureService(startCtx, resourceName, labels)
	if err != nil {
		return nil, fmt.Errorf("创建场景 Service 失败: %w", err)
	}
	if err := o.ensurePod(startCtx, resourceName, labels, config); err != nil {
		// Service 已创建但 Pod 创建失败 → 回滚 Service 避免悬挂资源
		_ = o.deleteService(context.Background(), resourceName)
		return nil, fmt.Errorf("创建场景 Pod 失败: %w", err)
	}

	conn, err := o.dialAndWaitReady(startCtx, clusterIP)
	if err != nil {
		// Pod 起来但 gRPC 不通 → 整体清理
		_ = o.deletePod(context.Background(), resourceName)
		_ = o.deleteService(context.Background(), resourceName)
		return nil, fmt.Errorf("等待场景 %s 就绪失败: %w", config.SceneCode, err)
	}

	pc := &podConnection{
		podName:     resourceName,
		serviceName: resourceName,
		conn:        conn,
	}

	o.mu.Lock()
	if existing, ok := o.pool[key]; ok {
		// 并发竞争：复用先入池的连接，丢弃当前
		o.mu.Unlock()
		_ = conn.Close()
		_ = o.deletePod(context.Background(), resourceName)
		_ = o.deleteService(context.Background(), resourceName)
		return o.wrapClient(existing), nil
	}
	o.pool[key] = pc
	o.mu.Unlock()

	return o.wrapClient(pc), nil
}

// ensureService 幂等创建 ClusterIP Service 并返回已分配的 ClusterIP，端口 50100。
// 返回 ClusterIP 而非 DNS：这样 sim-engine 无论跑在集群内（ClusterIP 路由）
// 还是集群外的 docker 容器（docker-desktop 允许访问 ClusterIP 段）都能直接拨号。
// CoreDNS 的 *.svc.cluster.local 只在集群内解析，集群外 docker 桥接网络无法解析。
func (o *K8sOrchestrator) ensureService(ctx context.Context, name string, labels map[string]string) (string, error) {
	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: o.cfg.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     corev1.ServiceTypeClusterIP,
			Selector: labels,
			Ports: []corev1.ServicePort{{
				Name:       "grpc",
				Port:       scenarioContainerPort,
				TargetPort: intstr.FromInt32(scenarioContainerPort),
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}
	created, err := o.clientset.CoreV1().Services(o.cfg.Namespace).Create(ctx, svc, metav1.CreateOptions{})
	if err != nil {
		if !apierrors.IsAlreadyExists(err) {
			return "", err
		}
		// 已存在：读取现有 Service 的 ClusterIP
		existing, getErr := o.clientset.CoreV1().Services(o.cfg.Namespace).Get(ctx, name, metav1.GetOptions{})
		if getErr != nil {
			return "", getErr
		}
		created = existing
	}
	if strings.TrimSpace(created.Spec.ClusterIP) == "" || created.Spec.ClusterIP == corev1.ClusterIPNone {
		return "", fmt.Errorf("场景 Service %s 未分配到 ClusterIP", name)
	}
	return created.Spec.ClusterIP, nil
}

// ensurePod 幂等创建场景算法容器 Pod。
func (o *K8sOrchestrator) ensurePod(ctx context.Context, name string, labels map[string]string, config Config) error {
	cpuReq := config.ResourceRequestCPU
	if cpuReq == "" {
		cpuReq = o.cfg.DefaultCPU
	}
	memReq := config.ResourceRequestMemory
	if memReq == "" {
		memReq = o.cfg.DefaultMemory
	}
	cpuQty, err := resource.ParseQuantity(cpuReq)
	if err != nil {
		return fmt.Errorf("解析 cpu request 失败: %w", err)
	}
	memQty, err := resource.ParseQuantity(memReq)
	if err != nil {
		return fmt.Errorf("解析 memory request 失败: %w", err)
	}
	cpuLimit, _ := resource.ParseQuantity(o.cfg.LimitCPU)
	memLimit, _ := resource.ParseQuantity(o.cfg.LimitMemory)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: o.cfg.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyOnFailure,
			// 关闭 K8s 默认的 Service env 注入：场景算法容器仅需 SCENARIO_LISTEN_ADDR /
			// SCENE_CODE 两个显式 env，namespace 内可能存在其它平台 Service（grpc-server
			// 等），它们的 *_HOST / *_PORT 环境变量会污染容器。服务发现走 DNS 即可。
			EnableServiceLinks: boolPtr(false),
			Containers: []corev1.Container{{
				Name:            scenarioContainerName,
				Image:           config.ContainerImageURL,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Env: []corev1.EnvVar{
					{
						Name:  "SCENARIO_LISTEN_ADDR",
						Value: fmt.Sprintf(":%d", scenarioContainerPort),
					},
					{
						// SCENE_CODE 是平台共享运行时镜像（scenarios/runtime:v1.0.0）按
						// 场景分发的关键参数；二进制读取该 env 选择要运行的场景定义。
						// 详见 sim-engine/scenarios/cmd/scenario/main.go。
						Name:  "SCENE_CODE",
						Value: config.SceneCode,
					},
				},
				Ports: []corev1.ContainerPort{{
					Name:          "grpc",
					ContainerPort: scenarioContainerPort,
					Protocol:      corev1.ProtocolTCP,
				}},
				Resources: corev1.ResourceRequirements{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    cpuQty,
						corev1.ResourceMemory: memQty,
					},
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    cpuLimit,
						corev1.ResourceMemory: memLimit,
					},
				},
				SecurityContext: &corev1.SecurityContext{
					AllowPrivilegeEscalation: boolPtr(false),
					Privileged:               boolPtr(false),
					RunAsNonRoot:             boolPtr(true),
					ReadOnlyRootFilesystem:   boolPtr(true),
				},
			}},
			SecurityContext: &corev1.PodSecurityContext{
				RunAsNonRoot: boolPtr(true),
			},
		},
	}

	if o.cfg.PullSecretName != "" {
		pod.Spec.ImagePullSecrets = []corev1.LocalObjectReference{{Name: o.cfg.PullSecretName}}
	}

	_, err = o.clientset.CoreV1().Pods(o.cfg.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return err
	}
	return nil
}

// dialAndWaitReady 主动拨号 + 重试 HealthCheck，直到通过或超时。
// 使用 Service 的 ClusterIP 直接拨号：集群内与集群外（docker-desktop 桥接）均可达。
func (o *K8sOrchestrator) dialAndWaitReady(ctx context.Context, clusterIP string) (*grpc.ClientConn, error) {
	target := fmt.Sprintf("%s:%d", clusterIP, scenarioContainerPort)
	conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	healthClient := simscenariov1.NewSimScenarioServiceClient(conn)
	ticker := time.NewTicker(healthCheckRetryInterval)
	defer ticker.Stop()

	for {
		probeCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		resp, healthErr := healthClient.HealthCheck(probeCtx, &simscenariov1.HealthCheckRequest{})
		cancel()
		if healthErr == nil && resp.GetStatus() == simscenariov1.HealthStatus_HEALTH_STATUS_SERVING {
			return conn, nil
		}

		select {
		case <-ctx.Done():
			_ = conn.Close()
			if healthErr != nil {
				return nil, fmt.Errorf("场景 gRPC 拨号重试超时: %w", healthErr)
			}
			return nil, fmt.Errorf("场景 gRPC 健康检查未通过，最后状态=%s", resp.GetStatus())
		case <-ticker.C:
		}
	}
}

// wrapClient 在 podConnection 上构建一个标准的 ScenarioClient。
// 注意：Close 不释放连接（连接归编排器拥有，由 DestroySession / EvictScene / Shutdown 管理）。
func (o *K8sOrchestrator) wrapClient(pc *podConnection) ScenarioClient {
	return &grpcScenarioClient{
		conn:   pc.conn,
		client: simscenariov1.NewSimScenarioServiceClient(pc.conn),
	}
}

// DestroySession 是 scene.Orchestrator 接口实现。删除会话名下所有场景的 Pod / Service，关闭连接。
func (o *K8sOrchestrator) DestroySession(ctx context.Context, sessionID string) error {
	o.mu.Lock()
	prefix := sessionID + "::"
	keys := make([]string, 0)
	for key := range o.pool {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	connections := make([]*podConnection, 0, len(keys))
	for _, key := range keys {
		connections = append(connections, o.pool[key])
		delete(o.pool, key)
	}
	o.mu.Unlock()

	var firstErr error
	for _, pc := range connections {
		if pc.conn != nil {
			_ = pc.conn.Close()
		}
		if err := o.deletePod(ctx, pc.podName); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := o.deleteService(ctx, pc.serviceName); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// EvictScene 是 scene.Orchestrator 接口实现。删除单个场景的 Pod / Service 与连接池条目，
// 使后续 StartScene 不会复用旧连接。
func (o *K8sOrchestrator) EvictScene(ctx context.Context, sessionID, sceneCode string) error {
	key := poolKey(sessionID, sceneCode)
	o.mu.Lock()
	pc, ok := o.pool[key]
	if ok {
		delete(o.pool, key)
	}
	o.mu.Unlock()

	var firstErr error
	if pc != nil {
		if pc.conn != nil {
			_ = pc.conn.Close()
		}
		if err := o.deletePod(ctx, pc.podName); err != nil && firstErr == nil {
			firstErr = err
		}
		if err := o.deleteService(ctx, pc.serviceName); err != nil && firstErr == nil {
			firstErr = err
		}
		return firstErr
	}

	// 池中找不到（可能 Pod 已不在池但仍残留在 K8s）：按命名约定兜底删除。
	resourceName := buildResourceName(sessionID, sceneCode)
	if err := o.deletePod(ctx, resourceName); err != nil && firstErr == nil {
		firstErr = err
	}
	if err := o.deleteService(ctx, resourceName); err != nil && firstErr == nil {
		firstErr = err
	}
	return firstErr
}

// StartIdleReaper 启动后台空闲场景 Pod 回收循环。
// 周期扫描 namespace 内由 sim-engine 管理的 Pod，删除满足以下条件者：
//  1. 不在当前连接池中（说明对应会话已被销毁或异常丢失）
//  2. Pod 已存在超过 IdleTTL（避免删除刚创建尚未入池的 Pod）
//
// 该循环在 ctx 被取消时退出。由 main.go 在 Engine 启动后立即调用。
func (o *K8sOrchestrator) StartIdleReaper(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(reaperInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				o.reapOnce(ctx)
			}
		}
	}()
}

// reapOnce 扫描一次孤儿场景 Pod 并清理。
// 出错只记录并跳过该 Pod，不打断循环。
func (o *K8sOrchestrator) reapOnce(ctx context.Context) {
	listOpts := metav1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=sim-engine,%s=scenario", labelManagedBy, labelComponent),
	}
	pods, err := o.clientset.CoreV1().Pods(o.cfg.Namespace).List(ctx, listOpts)
	if err != nil {
		return
	}

	now := time.Now()

	// 收集当前池中持有的 Pod 名集合，用于"不在池中"判定。
	o.mu.Lock()
	inPool := make(map[string]struct{}, len(o.pool))
	for _, pc := range o.pool {
		inPool[pc.podName] = struct{}{}
	}
	o.mu.Unlock()

	for _, pod := range pods.Items {
		if _, held := inPool[pod.Name]; held {
			continue
		}
		// Pod 必须超过 IdleTTL 才回收，避免误杀刚创建未入池的 Pod。
		age := now.Sub(pod.CreationTimestamp.Time)
		if age < o.cfg.IdleTTL {
			continue
		}
		_ = o.deletePod(ctx, pod.Name)
		_ = o.deleteService(ctx, pod.Name)
	}
}

// Shutdown 在进程退出时清理所有残留 Pod / Service。
// 双重保险：除已知池中的资源外，按 managed-by 标签兜底删除。
func (o *K8sOrchestrator) Shutdown(ctx context.Context) error {
	o.mu.Lock()
	connections := make([]*podConnection, 0, len(o.pool))
	for _, pc := range o.pool {
		connections = append(connections, pc)
	}
	o.pool = make(map[string]*podConnection)
	o.mu.Unlock()

	for _, pc := range connections {
		if pc.conn != nil {
			_ = pc.conn.Close()
		}
		_ = o.deletePod(ctx, pc.podName)
		_ = o.deleteService(ctx, pc.serviceName)
	}

	// 兜底：按 label 列出 namespace 内剩余场景资源并删除
	listOpts := metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=sim-engine,%s=scenario", labelManagedBy, labelComponent)}
	pods, err := o.clientset.CoreV1().Pods(o.cfg.Namespace).List(ctx, listOpts)
	if err == nil {
		for _, pod := range pods.Items {
			_ = o.deletePod(ctx, pod.Name)
		}
	}
	svcs, err := o.clientset.CoreV1().Services(o.cfg.Namespace).List(ctx, listOpts)
	if err == nil {
		for _, svc := range svcs.Items {
			_ = o.deleteService(ctx, svc.Name)
		}
	}
	return nil
}

// deletePod 幂等删除 Pod。
func (o *K8sOrchestrator) deletePod(ctx context.Context, name string) error {
	err := o.clientset.CoreV1().Pods(o.cfg.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// deleteService 幂等删除 Service。
func (o *K8sOrchestrator) deleteService(ctx context.Context, name string) error {
	err := o.clientset.CoreV1().Services(o.cfg.Namespace).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// buildRESTConfig 根据配置选择集群内 / 集群外 K8s 客户端。
func buildRESTConfig(cfg OrchestratorConfig) (*rest.Config, error) {
	if cfg.InCluster {
		return rest.InClusterConfig()
	}
	kubeconfig := cfg.KubeconfigPath
	if kubeconfig == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfig = home + "/.kube/config"
		}
	}
	return clientcmd.BuildConfigFromFlags("", kubeconfig)
}

// poolKey 生成连接池键。
func poolKey(sessionID, sceneCode string) string {
	return sessionID + "::" + sceneCode
}

// buildResourceName 生成 DNS-1123 合规的 Pod / Service 名称。
// 形如 scn-{sessionShort12}-{sceneNormalized30}，总长 ≤ 47。
func buildResourceName(sessionID, sceneCode string) string {
	short := sessionID
	if len(short) > 12 {
		short = short[len(short)-12:]
	}
	normalized := strings.ToLower(sceneCode)
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if len(normalized) > 30 {
		normalized = normalized[:30]
	}
	normalized = strings.Trim(normalized, "-")
	if normalized == "" {
		normalized = "scene"
	}
	return fmt.Sprintf("scn-%s-%s", strings.ToLower(short), normalized)
}

// sanitizeLabelValue 清洗为合法 K8s label value（≤ 63，DNS-1123 子集）。
func sanitizeLabelValue(value string) string {
	v := strings.ToLower(value)
	v = strings.ReplaceAll(v, "_", "-")
	if len(v) > 63 {
		v = v[:63]
	}
	v = strings.Trim(v, "-")
	if v == "" {
		v = "unknown"
	}
	return v
}

// boolPtr 返回 bool 指针。
func boolPtr(v bool) *bool {
	return &v
}
