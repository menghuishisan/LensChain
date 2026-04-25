# frontend.Dockerfile
# 链镜平台前端 Next.js 应用镜像（standalone 模式）
# 多阶段构建：node:20-alpine 构建 → node:20-alpine 运行
# 监听端口：3000
# 构建时变量：NEXT_PUBLIC_* 在构建阶段注入到客户端 Bundle，运行时无法修改

# ============================
# 依赖阶段
# ============================
FROM node:20-alpine AS deps

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund

# ============================
# 构建阶段
# ============================
FROM node:20-alpine AS builder

ARG NEXT_PUBLIC_API_BASE_URL
ARG NEXT_PUBLIC_WS_BASE_URL

WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .

ENV NEXT_TELEMETRY_DISABLED=1 \
    NEXT_PUBLIC_API_BASE_URL=${NEXT_PUBLIC_API_BASE_URL} \
    NEXT_PUBLIC_WS_BASE_URL=${NEXT_PUBLIC_WS_BASE_URL}

RUN npm run build

# ============================
# 运行阶段
# ============================
FROM node:20-alpine

ARG VERSION=dev

LABEL org.opencontainers.image.title="lenschain-frontend"
LABEL org.opencontainers.image.description="链镜平台前端 Next.js 应用"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/service="frontend"

ENV TZ=UTC \
    NODE_ENV=production \
    NEXT_TELEMETRY_DISABLED=1 \
    PORT=3000

RUN apk add --no-cache curl tzdata && \
    addgroup -S -g 1001 lenschain && \
    adduser -S -u 1001 -G lenschain lenschain

WORKDIR /app

# Next.js standalone 产物
COPY --from=builder --chown=lenschain:lenschain /app/.next/standalone ./
COPY --from=builder --chown=lenschain:lenschain /app/.next/static ./.next/static
COPY --from=builder --chown=lenschain:lenschain /app/public ./public

USER lenschain

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=15s \
    CMD curl -fsS http://localhost:3000 || exit 1

CMD ["node", "server.js"]
