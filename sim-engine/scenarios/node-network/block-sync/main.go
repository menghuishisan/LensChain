package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动区块同步与传播场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "block-sync"); err != nil {
		log.Fatal(err)
	}
}

