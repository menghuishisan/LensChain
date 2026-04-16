package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 DPoS 委托投票场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "dpos-voting"); err != nil {
		log.Fatal(err)
	}
}
