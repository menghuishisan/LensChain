package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 51% 算力攻击场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "51-percent-attack"); err != nil {
		log.Fatal(err)
	}
}
