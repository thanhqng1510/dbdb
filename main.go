package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/thanhqng1510/dbdb/conf"
	"github.com/thanhqng1510/dbdb/store"
)

func main() {
	cfg := conf.GetConfig(os.Args[1:])
	raftDataDir := path.Join("data", fmt.Sprintf("%s-raft", cfg.Id))

	storeCfg := store.Config{
		NodeID:    cfg.Id,
		RaftDir:   raftDataDir,
		RaftAddr:  "localhost:" + cfg.RaftPort, // Assuming localhost for Raft communication
		HttpAddr:  ":" + cfg.HttpPort,          // HTTP server listens on all interfaces
		Bootstrap: cfg.Bootstrap,
	}

	store, err := store.NewStore(storeCfg)
	if err != nil {
		log.Fatalf("Failed to create dbdb store: %v", err)
	}

	log.Printf("Starting dbdb node %s. Raft: %s, HTTP: %s. Bootstrap: %t. Join: %s",
		storeCfg.NodeID, storeCfg.RaftAddr, storeCfg.HttpAddr, storeCfg.Bootstrap, cfg.JoinAddr)

	// TODO: use joinAddr instead of join API
	// TODO: delete key
	// TODO: thread-safe

	// Start the HTTP server. This is a blocking call.
	if err := store.StartHttpServer(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
}
