package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动分布式存储 DHT 场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "dht-storage"); err != nil {
		log.Fatal(err)
	}
}

