/** @type {import('next').NextConfig} */
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
