package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动区块链结构与分叉场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "blockchain-structure"); err != nil {
		log.Fatal(err)
	}
}

