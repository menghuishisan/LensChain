// k8s_portforward.go
// 模块04 — 实验环境：通过 K8s API 的 SPDY 端口转发隧道把 Pod 端口包装为 net.Conn。
//
// 设计依据（必须遵守，不允许走"直拨 Pod IP / Service ClusterIP"等替代路径）：
//
//   docs/modules/09-部署与运维/02-基础设施设计.md §2.4 学生访问实验容器内服务通过
//   后端代理：后端使用 kubectl exec / kubectl port-forward / SPDY WebSocket 隧道
//   对学生浏览器提供 HTTP / WS 通道。
//
//   实验 Namespace 不创建 NodePort / LoadBalancer，避免占用宿主机端口或暴露到公网。
//
// 该实现等价于 `kubectl port-forward` 的工作机制：
//   1) 调用 Pod 的 portforward subresource，触发 K8s API server 升级到 SPDY；
//   2) 在 SPDY 连接上分别创建 error stream 与 data stream，K8s API server 把 data
//      stream 透明转发到目标 Pod 的 localhost:<port>；
//   3) 把 data stream 包装为 net.Conn 暴露给上层（如 gorilla/websocket Dialer）。
//
// 与其它备选方案的对比（已被本设计明确排除）：
//   - 直拨 Pod IP：仅在 backend 与 Pod 同一 overlay 网络时可达；本地开发（backend
//     跑宿主机 + docker-desktop K8s）/ 跨节点 / Pod IP 漂移等场景均失败。
//   - Service ClusterIP：与 Pod IP 相同的局限。
//   - NodePort / LoadBalancer：违反基础设施设计文档对实验 Namespace 的硬约束。
//
// 因此本仓库**只保留 SPDY 隧道一种**实现路径，不维护任何兼容/退化通道。

package experiment

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
	"k8s.io/apimachinery/pkg/util/httpstream"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

// DialPodPort 通过 K8s API 的 SPDY portforward 隧道连接到指定 Pod 端口。
//
// 调用方约定：
//   - (namespace, podName, port) 必须由 service 层结合学生身份做过访问校验后传入；
//     本函数仅做技术层封装，不做业务边界判断。
//   - 返回的 net.Conn 在不再使用时必须 Close，否则 SPDY 会话与 K8s API 资源会驻留。
func (k *k8sClient) DialPodPort(ctx context.Context, namespace, podName string, port int) (net.Conn, error) {
	if k == nil || k.clientset == nil || k.restConfig == nil {
		return nil, fmt.Errorf("k8s client is not initialized")
	}
	if namespace == "" || podName == "" || port <= 0 {
		return nil, fmt.Errorf("invalid portforward target: ns=%q pod=%q port=%d", namespace, podName, port)
	}

	// 1) 构造 portforward subresource URL：POST /api/v1/namespaces/<ns>/pods/<pod>/portforward
	req := k.clientset.CoreV1().RESTClient().
		Post().
		Resource("pods").
		Namespace(namespace).
		Name(podName).
		SubResource("portforward")

	// 2) 准备 SPDY upgrade transport（复用 backend 现有的 K8s 认证：kubeconfig 或
	//    in-cluster ServiceAccount）
	transport, upgrader, err := spdy.RoundTripperFor(k.restConfig)
	if err != nil {
		return nil, fmt.Errorf("create spdy roundtripper: %w", err)
	}

	// 3) 升级 HTTP/SPDY 连接
	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, http.MethodPost, req.URL())
	streamConn, _, err := dialer.Dial(portforward.PortForwardProtocolV1Name)
	if err != nil {
		return nil, fmt.Errorf("dial portforward subresource: %w", err)
	}

	// 4) requestID 用于把 data stream 与 error stream 在协议层关联起来；同一会话内唯一即可，
	//    复用 nano 时间戳满足该要求且避免引入随机源。
	requestID := strconv.FormatInt(time.Now().UnixNano(), 10)

	// 5) error stream：先建，K8s API 会在转发失败 / 连接异常时往这里写错误描述。
	errHeaders := http.Header{}
	errHeaders.Set(corev1.StreamType, corev1.StreamTypeError)
	errHeaders.Set(corev1.PortHeader, strconv.Itoa(port))
	errHeaders.Set(corev1.PortForwardRequestIDHeader, requestID)
	errStream, err := streamConn.CreateStream(errHeaders)
	if err != nil {
		_ = streamConn.Close()
		return nil, fmt.Errorf("create error stream: %w", err)
	}
	// 不向 error stream 写任何数据（K8s portforward 协议规定客户端只读不写）。
	_ = errStream.Close()

	// 6) data stream：真正的双向字节流，等价于 Pod 内 localhost:<port> 的 TCP 连接。
	dataHeaders := http.Header{}
	dataHeaders.Set(corev1.StreamType, corev1.StreamTypeData)
	dataHeaders.Set(corev1.PortHeader, strconv.Itoa(port))
	dataHeaders.Set(corev1.PortForwardRequestIDHeader, requestID)
	dataStream, err := streamConn.CreateStream(dataHeaders)
	if err != nil {
		_ = streamConn.Close()
		return nil, fmt.Errorf("create data stream: %w", err)
	}

	// 7) 异步消费 error stream，避免上游写入后阻塞 K8s API；错误内容只用作日志诊断。
	go func() {
		_, _ = io.Copy(io.Discard, errStream)
	}()

	return &portForwardConn{
		stream:     dataStream,
		streamConn: streamConn,
		remote: portForwardAddr{
			network:   "k8s-portforward",
			namespace: namespace,
			podName:   podName,
			port:      port,
		},
	}, nil
}

// portForwardConn 把 SPDY data stream 适配为 net.Conn。
//
// 上层（gorilla/websocket Dialer）期望 net.Conn 接口的 Read/Write/Close 与 SetDeadline。
// SPDY data stream 自身天然满足 Read/Write/Close；deadline 在 SPDY 层无对应原语，
// 因此 SetDeadline 系列方法返回 nil（即不支持），上层通过 ctx 控制生命周期。
type portForwardConn struct {
	stream     httpstream.Stream
	streamConn httpstream.Connection
	remote     portForwardAddr
	closed     atomic.Bool
}

// Read 从 Pod 端口读字节流。
func (c *portForwardConn) Read(p []byte) (int, error) { return c.stream.Read(p) }

// Write 把字节流写入 Pod 端口。
func (c *portForwardConn) Write(p []byte) (int, error) { return c.stream.Write(p) }

// Close 关闭 data stream 与底层 SPDY 会话；幂等。
func (c *portForwardConn) Close() error {
	if c.closed.Swap(true) {
		return nil
	}
	_ = c.stream.Close()
	return c.streamConn.Close()
}

// LocalAddr 返回 portForwardAddr 占位；本地址不映射真实 socket。
func (c *portForwardConn) LocalAddr() net.Addr { return portForwardAddr{network: "k8s-portforward"} }

// RemoteAddr 返回 (namespace, podName, port) 三元组的字符串描述，便于诊断。
func (c *portForwardConn) RemoteAddr() net.Addr { return c.remote }

// SetDeadline SPDY 不支持，上层 ctx 控制超时。
func (c *portForwardConn) SetDeadline(time.Time) error { return nil }

// SetReadDeadline SPDY 不支持。
func (c *portForwardConn) SetReadDeadline(time.Time) error { return nil }

// SetWriteDeadline SPDY 不支持。
func (c *portForwardConn) SetWriteDeadline(time.Time) error { return nil }

// portForwardAddr 是 net.Addr 的占位实现，仅用于诊断输出。
type portForwardAddr struct {
	network   string
	namespace string
	podName   string
	port      int
}

// Network 返回固定字符串 "k8s-portforward" 以与真实 TCP/UDP 区分。
func (a portForwardAddr) Network() string {
	if a.network == "" {
		return "k8s-portforward"
	}
	return a.network
}

// String 输出 ns/pod:port 形式，便于日志检索。
func (a portForwardAddr) String() string {
	if a.namespace == "" && a.podName == "" {
		return "k8s-portforward"
	}
	return fmt.Sprintf("%s/%s:%d", a.namespace, a.podName, a.port)
}
