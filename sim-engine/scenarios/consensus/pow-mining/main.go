package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 PoW 挖矿竞争场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "pow-mining"); err != nil {
		log.Fatal(err)
	}
}

