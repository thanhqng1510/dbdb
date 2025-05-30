package store

import (
	"fmt"
	"log"
	"net"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// Config holds the configuration for a Store.
type Config struct {
	NodeID    string
	RaftDir   string
	RaftAddr  string
	Bootstrap bool
}

// Store manages the Raft consensus and the key-value data.
type Store struct {
	nodeID   string
	raftDir  string
	raftAddr string
	raft     *raft.Raft
	data     *sync.Map
}

// NewStore creates and initializes a new Store.
func NewStore(cfg Config) (*Store, error) {
	s := &Store{
		nodeID:   cfg.NodeID,
		raftDir:  cfg.RaftDir,
		raftAddr: cfg.RaftAddr,
		data:     &sync.Map{},
	}

	if err := os.MkdirAll(s.raftDir, 0700); err != nil {
		return nil, fmt.Errorf("could not create raft directory %s: %w", s.raftDir, err)
	}

	// BoltDB store for logs and stable store.
	boltDBPath := path.Join(s.raftDir, "raft.db")
	boltStore, err := raftboltdb.NewBoltStore(boltDBPath)
	if err != nil {
		return nil, fmt.Errorf("could not create bolt store at %s: %w", boltDBPath, err)
	}

	// Snapshot store.
	snapshotPath := path.Join(s.raftDir, "snapshots")
	snapshots, err := raft.NewFileSnapshotStore(snapshotPath, 2, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("could not create snapshot store at %s: %w", snapshotPath, err)
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", s.raftAddr)
	if err != nil {
		return nil, fmt.Errorf("could not resolve raft address %s: %w", s.raftAddr, err)
	}

	var advertiseAddr *net.TCPAddr
	hostname := os.Getenv("HOSTNAME")
	if hostname != "" && os.Getenv("DOCKER_ENV") == "true" {
		// In Docker, use service name from docker-compose
		advertiseAddr, err = net.ResolveTCPAddr("tcp", hostname+":"+strconv.Itoa(tcpAddr.Port))
		if err != nil {
			return nil, fmt.Errorf("could not create advertise address: %w", err)
		}
		log.Printf("Running in Docker, advertising as %s", advertiseAddr)
	} else {
		// Use the same address for binding and advertising
		advertiseAddr = tcpAddr
	}

	transport, err := raft.NewTCPTransport(s.raftAddr, advertiseAddr, 10, time.Second*10, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("could not create tcp transport: %w", err)
	}

	raftCfg := raft.DefaultConfig()
	raftCfg.LocalID = raft.ServerID(s.nodeID)

	fsm := &kvFsm{data: s.data}
	r, err := raft.NewRaft(raftCfg, fsm, boltStore, boltStore, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("could not create raft instance: %w", err)
	}
	s.raft = r

	if cfg.Bootstrap {
		log.Printf("Bootstrapping cluster with node ID %s at %s", s.nodeID, transport.LocalAddr())
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(s.nodeID),
					Address: transport.LocalAddr(),
				},
			},
		}
		if err := s.raft.BootstrapCluster(configuration).Error(); err != nil {
			return nil, fmt.Errorf("could not bootstrap cluster: %w", err)
		}
	}
	return s, nil
}

// Set applies a command to set a key-value pair via Raft.
func (s *Store) Set(data []byte) error {
  if s.raft.State() != raft.Leader {
    return fmt.Errorf("not the leader")
  }
  
	future := s.raft.Apply(data, 500*time.Millisecond)
	if err := future.Error(); err != nil {
		return fmt.Errorf("could not apply set command via Raft: %w", err)
	}

	if fsmResponse := future.Response(); fsmResponse != nil {
		if fsmErr, ok := fsmResponse.(error); ok {
			return fmt.Errorf("FSM error on set command: %w", fsmErr)
		}
	}
	return nil
}

// Get retrieves a value by key from the store.
func (s *Store) Get(key string) interface{} {
	value, _ := s.data.Load(key)
	return value
}

// AddFollower adds a new node to the Raft cluster.
func (s *Store) AddFollower(followerId, followerAddr string) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader")
	}

	log.Printf("Handling join request for node %s at %s", followerId, followerAddr)
	if err := s.raft.AddVoter(raft.ServerID(followerId), raft.ServerAddress(followerAddr), 0, 0).Error(); err != nil {
		log.Printf("Failed to add voter %s (%s): %s", followerId, followerAddr, err)
		return err
	}
	return nil
}
