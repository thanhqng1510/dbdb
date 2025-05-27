package conf

import (
	"log"
)

type Config struct {
	Id       string
	HttpPort string
	RaftPort string
}

func GetConfig(args []string) Config {
	cfg := Config{}
	for i, arg := range args {
		if arg == "--node-id" {
			cfg.Id = args[i+2]
			i++
			continue
		}

		if arg == "--http-port" {
			cfg.HttpPort = args[i+2]
			i++
			continue
		}

		if arg == "--raft-port" {
			cfg.RaftPort = args[i+2]
			i++
			continue
		}
	}

	if cfg.Id == "" {
		log.Fatal("Missing required parameter: --node-id")
	}

	if cfg.RaftPort == "" {
		log.Fatal("Missing required parameter: --raft-port")
	}

	if cfg.HttpPort == "" {
		log.Fatal("Missing required parameter: --http-port")
	}

	return cfg
}