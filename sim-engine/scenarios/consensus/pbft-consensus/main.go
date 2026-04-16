package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 PBFT 三阶段共识场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "pbft-consensus"); err != nil {
		log.Fatal(err)
	}
}

