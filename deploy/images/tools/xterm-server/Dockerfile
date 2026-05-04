# tools/xterm-server/Dockerfile
# Web 终端服务（平台自研）
# 提供浏览器内终端访问，基于 xterm.js + WebSocket

FROM node:20-alpine AS builder

RUN apk add --no-cache python3 make g++

WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci --no-audit --no-fund
COPY tsconfig.json ./
COPY src/ ./src/
RUN npm run build
RUN npm prune --production

FROM node:20-alpine

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="xterm-server"
LABEL org.opencontainers.image.description="Web 终端服务"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"

ENV TZ=UTC

RUN apk add --no-cache curl tzdata libstdc++ && \
    addgroup -S lenschain && adduser -S lenschain -G lenschain

WORKDIR /app
COPY --from=builder --chown=lenschain:lenschain /app/dist ./dist
COPY --from=builder --chown=lenschain:lenschain /app/node_modules ./node_modules

USER lenschain

EXPOSE 3000

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=10s \
    CMD curl -fsS http://localhost:3000/healthz || exit 1

CMD ["node", "dist/server.js"]
