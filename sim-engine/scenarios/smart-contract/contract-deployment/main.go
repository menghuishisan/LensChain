package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动合约部署流程场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "contract-deployment"); err != nil {
		log.Fatal(err)
	}
}
