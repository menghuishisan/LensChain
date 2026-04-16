package main

import (
	"context"
	"log"

	"github.com/lenschain/sim-engine/scenarios/internal/launcher"
)

// main 启动状态树（MPT）场景服务。
func main() {
	if err := launcher.MustRunByCode(context.Background(), "mpt-trie"); err != nil {
		log.Fatal(err)
	}
}

