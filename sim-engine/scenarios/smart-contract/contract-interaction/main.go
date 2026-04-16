package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动合约间调用场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "contract-interaction"); err != nil {
		log.Fatal(err)
	}
}
