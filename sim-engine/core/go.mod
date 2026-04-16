module github.com/lenschain/sim-engine/core

go 1.24.0

toolchain go1.24.3

require github.com/lenschain/sim-engine/proto/gen/go v0.0.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/goccy/go-json v0.10.3 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.8 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require (
	github.com/minio/minio-go/v7 v7.0.80
	golang.org/x/net v0.49.0
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260120221211-b8f7ae30c516 // indirect
	google.golang.org/grpc v1.80.0
	google.golang.org/protobuf v1.36.11 // indirect
)

replace github.com/lenschain/sim-engine/proto/gen/go => ../proto/gen/go
