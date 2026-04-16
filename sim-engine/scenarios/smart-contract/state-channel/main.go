package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动状态通道场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "state-channel"); err != nil {
		log.Fatal(err)
	}
}
