package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动 Raft 领导选举场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "raft-election"); err != nil {
		log.Fatal(err)
	}
}

