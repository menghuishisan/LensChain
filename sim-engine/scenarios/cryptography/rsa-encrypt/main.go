package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 RSA 加密解密场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "rsa-encrypt"); err != nil {
		log.Fatal(err)
	}
}

