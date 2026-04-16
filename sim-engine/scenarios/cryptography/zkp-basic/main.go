package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动零知识证明原理场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "zkp-basic"); err != nil {
		log.Fatal(err)
	}
}

