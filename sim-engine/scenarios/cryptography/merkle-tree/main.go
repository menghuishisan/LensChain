package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 Merkle 树构建验证场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "merkle-tree"); err != nil {
		log.Fatal(err)
	}
}

