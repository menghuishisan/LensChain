// safe_client.go
// SSRF 安全的 HTTP 客户端
// 用于 SSO 连接测试等需要访问外部 URL 的场景
// 防止服务端请求伪造（SSRF）攻击：仅允许 HTTPS、拒绝私有/回环 IP

package httpclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var (
	// ErrUnsafeURL 不安全的 URL（非 HTTPS 或指向内网地址）
	ErrUnsafeURL = errors.New("不安全的URL：仅允许HTTPS且不可指向内网地址")
	// ErrPrivateIP 目标地址为私有/回环/链路本地 IP
	ErrPrivateIP = errors.New("目标地址为内网地址，禁止访问")
	// ErrHTTPScheme 非 HTTPS 协议
	ErrHTTPScheme = errors.New("仅允许HTTPS协议")
)

// SafeGet 安全的 HTTP GET 请求
// 1. 仅允许 HTTPS 协议
// 2. DNS 解析后校验 IP，拒绝私有/回环/链路本地地址
// 3. 禁止跟随重定向到不安全地址
// 4. 10 秒超时
func SafeGet(ctx context.Context, rawURL string) (*http.Response, error) {
	if err := validateHTTPSURL(rawURL); err != nil {
		return nil, err
	}
	client := newSafeClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%w", err)
	}

	return client.Do(req)
}

// SafePostForm 安全的表单 POST 请求
func SafePostForm(ctx context.Context, rawURL string, form url.Values) (*http.Response, error) {
	if err := validateHTTPSURL(rawURL); err != nil {
		return nil, err
	}
	client := newSafeClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	return client.Do(req)
}

// SafeGetWithBearer 安全的 Bearer 鉴权 GET 请求
func SafeGetWithBearer(ctx context.Context, rawURL, token string) (*http.Response, error) {
	if err := validateHTTPSURL(rawURL); err != nil {
		return nil, err
	}
	client := newSafeClient()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("创建请求失败：%w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)

	return client.Do(req)
}

func newSafeClient() *http.Client {
	// 校验协议
	return &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			DialContext: safeDialContext,
			// 禁止自动跟随重定向（防止重定向到内网）
			// 通过 CheckRedirect 控制
		},
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 最多跟随 3 次重定向
			if len(via) >= 3 {
				return errors.New("重定向次数过多")
			}
			// 重定向目标也必须是 HTTPS
			if req.URL.Scheme != "https" {
				return ErrHTTPScheme
			}
			return nil
		},
	}
}

// validateHTTPSURL 校验仅允许HTTPS
func validateHTTPSURL(rawURL string) error {
	if !strings.HasPrefix(rawURL, "https://") {
		return ErrHTTPScheme
	}
	return nil
}

// safeDialContext 安全的 TCP 连接建立
// DNS 解析后校验目标 IP，拒绝私有/回环/链路本地地址
func safeDialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("解析地址失败：%w", err)
	}

	// DNS 解析
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return nil, fmt.Errorf("DNS解析失败：%w", err)
	}

	// 校验所有解析结果
	for _, ip := range ips {
		if isPrivateIP(ip.IP) {
			return nil, ErrPrivateIP
		}
	}

	// 使用第一个安全的 IP 建立连接
	dialer := &net.Dialer{Timeout: 5 * time.Second}
	return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
}

// isPrivateIP 判断 IP 是否为私有/回环/链路本地地址
func isPrivateIP(ip net.IP) bool {
	// 回环地址 127.0.0.0/8, ::1
	if ip.IsLoopback() {
		return true
	}
	// 链路本地 169.254.0.0/16, fe80::/10
	if ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() {
		return true
	}
	// 未指定地址 0.0.0.0, ::
	if ip.IsUnspecified() {
		return true
	}

	// RFC 1918 私有地址
	privateRanges := []struct {
		network *net.IPNet
	}{
		{mustParseCIDR("10.0.0.0/8")},
		{mustParseCIDR("172.16.0.0/12")},
		{mustParseCIDR("192.168.0.0/16")},
		// IPv6 私有地址
		{mustParseCIDR("fc00::/7")},
		// AWS/云元数据地址
		{mustParseCIDR("169.254.169.254/32")},
	}

	for _, r := range privateRanges {
		if r.network.Contains(ip) {
			return true
		}
	}

	return false
}

// mustParseCIDR 解析 CIDR，失败时 panic（仅用于初始化常量）
func mustParseCIDR(cidr string) *net.IPNet {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(fmt.Sprintf("无效的CIDR: %s", cidr))
	}
	return network
}
