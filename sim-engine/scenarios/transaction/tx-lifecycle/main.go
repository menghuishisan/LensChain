package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动交易生命周期场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "tx-lifecycle"); err != nil {
		log.Fatal(err)
	}
}

