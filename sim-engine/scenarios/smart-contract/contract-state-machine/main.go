package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动智能合约状态机场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "contract-state-machine"); err != nil {
		log.Fatal(err)
	}
}
