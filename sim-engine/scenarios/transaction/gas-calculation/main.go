package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 Gas 计算与优化场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "gas-calculation"); err != nil {
		log.Fatal(err)
	}
}

