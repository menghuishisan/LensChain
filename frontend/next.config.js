/** @type {import('next').NextConfig} */
const nextConfig = {
  reactStrictMode: true,
  output: 'standalone',
  transpilePackages: ['@lenschain/sim-engine-renderers'],
};

module.exports = nextConfig;
