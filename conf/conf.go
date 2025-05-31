package conf

import (
	"flag"
	"os"
)

// Config holds the node configuration.
type Config struct {
	// Node ID
	Id        string

	// Port for Raft communication
	RaftPort  string
	
	// Port for HTTP API
	HttpPort  string

	// Address of an existing node within a cluster to join (e.g., "localhost:8221")
	JoinAddr  string

	// If true, bootstrap a new cluster (should only be true for the first node),
	// This cannot be used with JoinAddr
	Bootstrap bool
}

// GetConfig parses command-line arguments and returns the configuration.
func GetConfig(args []string) Config {
	var cfg Config
	fs := flag.NewFlagSet("dbdb", flag.ExitOnError)
	fs.StringVar(&cfg.Id, "node-id", "", "Node ID (required)")
	fs.StringVar(&cfg.RaftPort, "raft-port", "", "Raft communication port (required)")
	fs.StringVar(&cfg.HttpPort, "http-port", "", "HTTP API port (required)")
	fs.StringVar(&cfg.JoinAddr, "join", "", "Address of a leader node to join (HTTP API address)")
	fs.BoolVar(&cfg.Bootstrap, "bootstrap", false, "Bootstrap as the first node in a new cluster")

	fs.Parse(args)

	if cfg.Bootstrap && cfg.JoinAddr != "" {
		fs.Usage() // Print usage information
		os.Stderr.WriteString("Error: --bootstrap cannot be used with --join\n")
		os.Exit(1)
	}

	// Check for mandatory fields
	if cfg.Id == "" {
		fs.Usage()
		os.Stderr.WriteString("Error: --node-id is required\n")
		os.Exit(1)
	}
	if cfg.RaftPort == "" {
		fs.Usage()
		os.Stderr.WriteString("Error: --raft-port is required\n")
		os.Exit(1)
	}
	if cfg.HttpPort == "" {
		flag.Usage() // Print usage information
		os.Stderr.WriteString("Error: --http-port is required\n")
		os.Exit(1)
	}
	return cfg
}