package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 SHA-256 哈希过程场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "sha256-hash"); err != nil {
		log.Fatal(err)
	}
}

