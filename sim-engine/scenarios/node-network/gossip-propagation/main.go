package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 Gossip 消息传播场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "gossip-propagation"); err != nil {
		log.Fatal(err)
	}
}

