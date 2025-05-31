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

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("Failed to get hostname: %v", err)
	}
	if hostname == "" {
		log.Fatal("Hostname is empty. Please set the hostname for this machine.")
	}

	storeCfg := store.Config{
		NodeID:            cfg.Id,
		RaftDir:           raftDataDir,
		RaftAddr:          "0.0.0.0:"+cfg.RaftPort,
		RaftAdvertiseAddr: hostname + ":" + cfg.RaftPort,
		Bootstrap:         cfg.Bootstrap,
		JoinAddr:          cfg.JoinAddr,
	}

	store, err := store.NewStore(storeCfg)
	if err != nil {
		log.Fatalf("Failed to create dbdb store: %v", err)
	}
	
	log.Printf("Starting dbdb node %s. Raft: %s. Bootstrap: %t. Join: %s",
		storeCfg.NodeID, storeCfg.RaftAddr, storeCfg.Bootstrap, storeCfg.JoinAddr)

	// TODO: thread-safe
	// TODO: does this scale
	// TODO: allow to set / delete on any nodes
	// TODO: option to get data from all nodes or just the leader

	httpServer := http.NewServer(":"+cfg.HttpPort, store)
	if err := httpServer.Start(); err != nil {
		log.Fatalf("HTTP server failed: %v", err)
	}
	log.Printf("HTTP server started on port %s", cfg.HttpPort)
}
