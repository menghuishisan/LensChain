package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 PoS 验证者选举场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "pos-validator"); err != nil {
		log.Fatal(err)
	}
}

