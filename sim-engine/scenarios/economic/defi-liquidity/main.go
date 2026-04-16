package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 DeFi 流动性池场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "defi-liquidity"); err != nil {
		log.Fatal(err)
	}
}
