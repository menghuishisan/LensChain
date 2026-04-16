package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 PoS 质押经济场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "pos-staking"); err != nil {
		log.Fatal(err)
	}
}
