package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 ECDSA 签名验签场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "ecdsa-sign"); err != nil {
		log.Fatal(err)
	}
}

