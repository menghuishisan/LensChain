package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 Token 经济模型场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "token-economics"); err != nil {
		log.Fatal(err)
	}
}
