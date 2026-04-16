package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动交易排序与 MEV 场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "tx-ordering-mev"); err != nil {
		log.Fatal(err)
	}
}

