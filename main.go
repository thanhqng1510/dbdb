package main

import (
	"fmt"
	"log"
	"os"
	"path"

	"github.com/thanhqng1510/dbdb/conf"
	"github.com/thanhqng1510/dbdb/http"
	"github.com/thanhqng1510/dbdb/store"
)

func main() {
	cfg := conf.GetConfig(os.Args[1:])
	raftDataDir := path.Join("data", fmt.Sprintf("%s-raft", cfg.Id))

	storeCfg := store.Config{
		NodeID:    cfg.Id,
		RaftDir:   raftDataDir,
		RaftAddr:  "0.0.0.0:"+cfg.RaftPort,     // Listen on all interfaces
		Bootstrap: cfg.Bootstrap,
	}

	store, err := store.NewStore(storeCfg)
	if err != nil {
		log.Fatalf("Failed to create dbdb store: %v", err)
	}
	
	log.Printf("Starting dbdb node %s. Raft: %s. Bootstrap: %t. Join: %s",
		storeCfg.NodeID, storeCfg.RaftAddr, storeCfg.Bootstrap, cfg.JoinAddr)

	// TODO: use joinAddr instead of join API
	// TODO: delete key
	// TODO: thread-safe
	// TODO: update README with Dockerfile and docker-compose
	// TODO: in docker-compose, no need different ports for each node
	// TODO: does this scale
	// TODO: allow to set / delete on any nodes

	httpServer := http.NewServer(":"+cfg.HttpPort, store)
	if err := httpServer.Start(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
	log.Printf("HTTP server started on port %s", cfg.HttpPort)
}
