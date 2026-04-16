package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动网络分区与恢复场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "network-partition"); err != nil {
		log.Fatal(err)
	}
}

