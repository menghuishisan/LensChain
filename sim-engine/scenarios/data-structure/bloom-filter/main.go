package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动布隆过滤器场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "bloom-filter"); err != nil {
		log.Fatal(err)
	}
}

