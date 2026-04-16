package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动整数溢出攻击场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "integer-overflow"); err != nil {
		log.Fatal(err)
	}
}
