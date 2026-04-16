package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 PBFT 拜占庭攻击场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "pbft-byzantine"); err != nil {
		log.Fatal(err)
	}
}
