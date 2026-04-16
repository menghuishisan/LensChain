package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动链上治理投票场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "governance-voting"); err != nil {
		log.Fatal(err)
	}
}
