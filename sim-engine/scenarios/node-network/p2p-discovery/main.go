package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 P2P 网络发现与路由场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "p2p-discovery"); err != nil {
		log.Fatal(err)
	}
}

