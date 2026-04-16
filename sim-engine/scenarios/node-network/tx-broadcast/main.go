package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动交易广播与打包场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "tx-broadcast"); err != nil {
		log.Fatal(err)
	}
}

