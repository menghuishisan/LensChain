// k8s_portforward.go
// 模块：sim-engine/core/internal/scene
// 文件职责：通过 K8s API 的 SPDY portforward 隧道，把场景 Pod 的 gRPC 端口包装为 net.Conn。
//
// 适用场景（与 docs/modules/04-实验环境/06.1-场景算法容器编排设计.md 一致）：
//   - 生产环境 (in_cluster=true)：sim-engine 跑在 K8s Pod 内，节点上 kube-proxy 路由
//     ClusterIP 段；orchestrator 走 net.Dial 直拨 ClusterIP:50100（不进入本文件）。
//   - 开发环境 (in_cluster=false)：sim-engine 跑在 docker-compose 桥接网络，与 docker-desktop
//     K8s 的 ClusterIP 段不连通；改走本文件 SPDY 隧道——与 backend tool proxy 用同一套
//     技术（参见 backend/internal/service/experiment/k8s_portforward.go）。
//
// 该实现等价于 `kubectl port-forward`：
//   1) POST /api/v1/namespaces/<ns>/pods/<pod>/portforward 升级到 SPDY；
//   2) 在 SPDY 连接上分别打开 error stream 与 data stream，K8s API server 把 data
//      stream 透明转发到目标 Pod 的 localhost:<port>；
//   3) 把 data stream 包装成 net.Conn 暴露给 grpc.WithContextDialer。

package scene

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// dialPodPortViaSPDY 通过 K8s API 的 SPDY portforward 隧道，连接到指定 Pod 端口。
//
// 调用前须知：
//   - Pod 必须已处于 Running 状态，否则 K8s API server 直接拒绝 portforward 请求。
//     调用方需先调用 waitPodRunning 等待 Pod 就绪。
//   - 返回的 net.Conn 不再使用时必须 Close，否则 SPDY 会话与 K8s API 资源会驻留。
func (o *K8sOrchestrator) dialPodPortViaSPDY(ctx context.Context, podName string, port int32) (net.Conn, error) {
	if o == nil || o.clientset == nil || o.restConfig == nil {
		return nil, fmt.Errorf("k8s client 未初始化")
	}
	if podName == "" || port <= 0 {
		return nil, fmt.Errorf("非法 portforward 目标: pod=%q port=%d", podName, port)
	}

	req := o.clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(o.cfg.Namespace).
		Name(podName).
		SubResource("portforward")

	transport, upgrader, err := spdy.RoundTripperFor(o.restConfig)
	if err != nil {
		return nil, fmt.Errorf("构造 spdy roundtripper 失败: %w", err)
	}

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	streamConn, _, err := dialer.Dial(portforward.PortForwardProtocolV1Name)
	if err != nil {
		return nil, fmt.Errorf("dial portforward subresource 失败: %w", err)
	}

	requestID := strconv.FormatInt(time.Now().UnixNano(), 10)

	// error stream：先建，K8s API 在转发失败 / 连接异常时往这里写错误描述。
	errHeaders := http.Header{}
	errHeaders.Set(corev1.StreamType, corev1.StreamTypeError)
	errHeaders.Set(corev1.PortHeader, strconv.Itoa(int(port)))
	errHeaders.Set(corev1.PortForwardRequestIDHeader, requestID)
	errStream, err := streamConn.CreateStream(errHeaders)
	if err != nil {
		_ = streamConn.Close()
		return nil, fmt.Errorf("创建 error stream 失败: %w", err)
	}
	_ = errStream.Close() // K8s portforward 协议规定客户端只读不写

	// data stream：双向字节流，等价于 Pod 内 localhost:<port> 的 TCP 连接。
	dataHeaders := http.Header{}
	dataHeaders.Set(corev1.StreamType, corev1.StreamTypeData)
	dataHeaders.Set(corev1.PortHeader, strconv.Itoa(int(port)))
	dataHeaders.Set(corev1.PortForwardRequestIDHeader, requestID)
	dataStream, err := streamConn.CreateStream(dataHeaders)
	if err != nil {
		_ = streamConn.Close()
		return nil, fmt.Errorf("创建 data stream 失败: %w", err)
	}

	// 异步消费 error stream，避免上游写入后阻塞 K8s API；错误内容只用作日志诊断。
	go func() { _, _ = io.Copy(io.Discard, errStream) }()

	return &portForwardConn{
		stream:     dataStream,
		streamConn: streamConn,
		remote: portForwardAddr{
			network:   "k8s-portforward",
			namespace: o.cfg.Namespace,
			podName:   podName,
			port:      int(port),
		},
	}, nil
}

// waitPodRunning 轮询等待 Pod 进入 Running 阶段。
//
// 用于 portforward 路径：SPDY 子资源要求 Pod phase=Running 才能转发。
// 直拨 ClusterIP 路径不依赖此函数（HealthCheck 重试隐式等待）。
func (o *K8sOrchestrator) waitPodRunning(ctx context.Context, podName string) error {
	const pollInterval = 300 * time.Millisecond
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		pod, err := o.clientset.CoreV1().Pods(o.cfg.Namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("查询 Pod 状态失败: %w", err)
			}
			// NotFound 阶段（刚 Create 还没出现在 cache）：继续轮询。
		} else if pod.Status.Phase == corev1.PodRunning {
			return nil
		} else if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded {
			return fmt.Errorf("Pod 已退出，phase=%s", pod.Status.Phase)
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("等待 Pod Running 超时: %w", ctx.Err())
		case <-ticker.C:
		}
	}
}

// portForwardConn 把 SPDY data stream 适配为 net.Conn（grpc.WithContextDialer 期望的接口）。
//
// SetDeadline 系列必须真实兑现，否则 gRPC client（包括 keepalive PING、ctx.Deadline 传播）
// 没法在 SPDY 隧道死亡时把阻塞的 Read/Write 拉回来——会导致 sim-engine 的 runtime.opMu
// 被永久持有，前端的 play/pause/step 等控制指令排队失效（生产事故级）。
//
// 由于 httpstream.Stream 本身没有 deadline 支持，这里采用 timer-on-fire-close 模式：
//   - 调用 SetXxxDeadline(t) 时调度一个 time.Timer
//   - 到期触发 c.Close() —— 关掉整个 streamConn，让所有 pending 的 Read/Write 立刻返回错误
//   - gRPC client 检测到 conn 不可用后会通过 grpc.WithContextDialer 重新拨号一条新的
//     portforward 隧道（Pod 不动），自然恢复
//
// 与"关流"语义匹配 net.Conn 接口契约（deadline 触发后此连接不再可用，调用方必须重连）。
type portForwardConn struct {
	stream     httpstream.Stream
	streamConn httpstream.Connection
	remote     portForwardAddr
	closed     atomic.Bool

	// Read / Write 方向各持有一个独立的 deadline timer，互不干扰。
	readDeadlineTimer  atomic.Pointer[time.Timer]
	writeDeadlineTimer atomic.Pointer[time.Timer]
}

func (c *portForwardConn) Read(p []byte) (int, error)  { return c.stream.Read(p) }
func (c *portForwardConn) Write(p []byte) (int, error) { return c.stream.Write(p) }

func (c *portForwardConn) Close() error {
	if !c.closed.CompareAndSwap(false, true) {
		return nil
	}
	if t := c.readDeadlineTimer.Swap(nil); t != nil {
		t.Stop()
	}
	if t := c.writeDeadlineTimer.Swap(nil); t != nil {
		t.Stop()
	}
	_ = c.stream.Close()
	return c.streamConn.Close()
}

func (c *portForwardConn) LocalAddr() net.Addr  { return portForwardAddr{network: "k8s-portforward"} }
func (c *portForwardConn) RemoteAddr() net.Addr { return c.remote }

func (c *portForwardConn) SetDeadline(t time.Time) error {
	if err := c.setDirectionalDeadline(&c.readDeadlineTimer, t); err != nil {
		return err
	}
	return c.setDirectionalDeadline(&c.writeDeadlineTimer, t)
}

func (c *portForwardConn) SetReadDeadline(t time.Time) error {
	return c.setDirectionalDeadline(&c.readDeadlineTimer, t)
}

func (c *portForwardConn) SetWriteDeadline(t time.Time) error {
	return c.setDirectionalDeadline(&c.writeDeadlineTimer, t)
}

// setDirectionalDeadline 是 SetReadDeadline / SetWriteDeadline 的共享实现。
//
//   - 传零值表示清除 deadline；
//   - 已过期的 deadline 会立即触发 Close；
//   - 未来 deadline 会调度一次性 timer，到期触发 Close。
//     gRPC 在 RPC 结束后会重置 deadline 为零，timer 会被取消，无副作用。
func (c *portForwardConn) setDirectionalDeadline(slot *atomic.Pointer[time.Timer], t time.Time) error {
	if old := slot.Swap(nil); old != nil {
		old.Stop()
	}
	if c.closed.Load() {
		return net.ErrClosed
	}
	if t.IsZero() {
		return nil
	}
	duration := time.Until(t)
	if duration <= 0 {
		// 已经过期：立即关闭。
		_ = c.Close()
		return nil
	}
	timer := time.AfterFunc(duration, func() {
		_ = c.Close()
	})
	slot.Store(timer)
	return nil
}

// portForwardAddr 是 net.Addr 的占位实现，仅用于诊断输出。
type portForwardAddr struct {
	network   string
	namespace string
	podName   string
	port      int
}

func (a portForwardAddr) Network() string {
	if a.network == "" {
		return "k8s-portforward"
	}
	return a.network
}

func (a portForwardAddr) String() string {
	if a.namespace == "" && a.podName == "" {
		return "k8s-portforward"
	}
	return fmt.Sprintf("%s/%s:%d", a.namespace, a.podName, a.port)
}
