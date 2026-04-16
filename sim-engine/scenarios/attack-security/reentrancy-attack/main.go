package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动重入攻击场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "reentrancy-attack"); err != nil {
		log.Fatal(err)
	}
}
