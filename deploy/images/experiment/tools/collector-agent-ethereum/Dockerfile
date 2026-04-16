FROM golang:1.24-alpine AS builder

WORKDIR /src/sim-engine/core
COPY sim-engine/core/go.mod sim-engine/core/go.sum ./
COPY sim-engine/proto/gen/go ../proto/gen/go
RUN go mod download

COPY sim-engine/core/ ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/collector-agent ./cmd/collector-agent

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /out/collector-agent /usr/local/bin/collector-agent
ENTRYPOINT ["/usr/local/bin/collector-agent"]
