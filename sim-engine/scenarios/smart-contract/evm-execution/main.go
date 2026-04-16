package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 EVM 执行步进场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "evm-execution"); err != nil {
		log.Fatal(err)
	}
}
