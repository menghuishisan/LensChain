package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动节点负载均衡场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "node-load-balance"); err != nil {
		log.Fatal(err)
	}
}

