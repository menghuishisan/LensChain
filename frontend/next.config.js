/** @type {import('next').NextConfig} */

// 工具反代 URL 策略（详见 backend/internal/handler/experiment/tool_proxy.go）：
//
// 后端把 instance_containers.proxy_url 签发为相对路径 /instance/<id>/<kind>/，
// 前端通过 NEXT_PUBLIC_TOOL_PROXY_BASE_URL 决定 iframe / WS 的最终 origin：
//   - 生产环境：BASE_URL 留空，相对路径走同一 Ingress 兜底（HTML / 子资源 / WS 同源）；
//   - 本地开发：BASE_URL=http://localhost:8080，iframe 直连后端，绕开 Next dev 对
//     `/instance/<id>/<kind>/` 末尾斜杠与 WS upgrade 的路径归一化限制（参考前端
//     services/experimentToolProxy.ts 与 components/business/WebIDEPanel.tsx）。
//
// 这意味着 next.config.js 不再为 /instance/* 配 rewrite——dev 与 prod 共用同一种 URL
// 拼装方式，BASE_URL 是部署配置而非业务分支。

const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  experimental: {
    // 优化 barrel 导入：仅编译实际用到的图标，避免 lucide-react 把全部图标拉进 dev chunk。
    optimizePackageImports: ['lucide-react'],
  },
  // 注：@lenschain/sim-engine-renderers 已通过自身 tsc 预编译为 dist/*.js，
  // 无需再让 Next.js 重复转译，去掉 transpilePackages 与 extensionAlias 后 dev 编译显著变快。
};

module.exports = nextConfig;
