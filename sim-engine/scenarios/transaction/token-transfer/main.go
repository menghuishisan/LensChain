package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 Token 转账流转场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "token-transfer"); err != nil {
		log.Fatal(err)
	}
}

