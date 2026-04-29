/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  transpilePackages: ['@lenschain/sim-engine-renderers'],
  webpack: (config) => {
    // sim-engine/renderers 使用 NodeNext 风格的 .js 扩展引用 TS 源文件，
    // 需要让 Webpack 将 .js 解析回 .ts 才能在 Next.js 中编译。
    config.resolve.extensionAlias = {
      ...(config.resolve.extensionAlias ?? {}),
      '.js': ['.ts', '.tsx', '.js'],
    };
    return config;
  },
};

module.exports = nextConfig;
