# tools/burpsuite-ce/Dockerfile
# Burp Suite Community Edition Web 安全测试工具（自建镜像）
# CTF Web 方向安全测试

FROM eclipse-temurin:17-jre-alpine

ARG VERSION=v1.0.0
LABEL org.opencontainers.image.title="burpsuite-ce"
LABEL org.opencontainers.image.description="Burp Suite Community Edition Web 安全测试工具"
LABEL org.opencontainers.image.version="${VERSION}"
LABEL org.opencontainers.image.vendor="LensChain"
LABEL lenschain.io/ecosystem="general"
LABEL lenschain.io/category="tools"
LABEL lenschain.io/scope="ctf"

ENV TZ=UTC

RUN apk add --no-cache curl tzdata

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --retries=3 --start-period=60s \
    CMD curl -fsS http://localhost:8080/ || exit 1

CMD ["java", "-jar", "/opt/burpsuite/burpsuite_community.jar"]
