package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 Gas 费市场场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "gas-market"); err != nil {
		log.Fatal(err)
	}
}
