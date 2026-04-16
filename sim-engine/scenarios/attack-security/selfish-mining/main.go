package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动自私挖矿场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "selfish-mining"); err != nil {
		log.Fatal(err)
	}
}
