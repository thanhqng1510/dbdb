package store

import (
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path"
	"sync"
	"time"

	"github.com/hashicorp/raft"
	raftboltdb "github.com/hashicorp/raft-boltdb"
)

// Config holds the configuration for a Store.
type Config struct {
	NodeID            string
	RaftDir           string
	RaftAddr          string
	RaftAdvertiseAddr string
	Bootstrap         bool
	JoinAddr          string
}

// Store manages the Raft consensus and the key-value data.
type Store struct {
	config Config
	raft   *raft.Raft
	data   *sync.Map
}

// NewStore creates and initializes a new Store.
func NewStore(cfg Config) (*Store, error) {
	s := &Store{
		config: cfg,
		data:   &sync.Map{},
	}

	if err := os.MkdirAll(s.config.RaftDir, 0700); err != nil {
		return nil, fmt.Errorf("could not create raft directory %s: %w", s.config.RaftDir, err)
	}

	// BoltDB store for logs and stable store.
	boltDBPath := path.Join(s.config.RaftDir, "raft.db")
	boltStore, err := raftboltdb.NewBoltStore(boltDBPath)
	if err != nil {
		return nil, fmt.Errorf("could not create bolt store at %s: %w", boltDBPath, err)
	}

	// Snapshot store.
	snapshotPath := path.Join(s.config.RaftDir, "snapshots")
	snapshots, err := raft.NewFileSnapshotStore(snapshotPath, 2, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("could not create snapshot store at %s: %w", snapshotPath, err)
	}

	advertiseAddr, err := net.ResolveTCPAddr("tcp", s.config.RaftAdvertiseAddr)
	if err != nil {
		return nil, fmt.Errorf("could not resolve raft advertise address %s: %w", s.config.RaftAdvertiseAddr, err)
	}

	transport, err := raft.NewTCPTransport(s.config.RaftAddr, advertiseAddr, 10, time.Second*10, os.Stderr)
	if err != nil {
		return nil, fmt.Errorf("could not create tcp transport: %w", err)
	}

	raftCfg := raft.DefaultConfig()
	raftCfg.LocalID = raft.ServerID(s.config.NodeID)

	fsm := &kvFsm{data: s.data}
	r, err := raft.NewRaft(raftCfg, fsm, boltStore, boltStore, snapshots, transport)
	if err != nil {
		return nil, fmt.Errorf("could not create raft instance: %w", err)
	}
	s.raft = r

	if s.config.Bootstrap {
		hasState, err := raft.HasExistingState(boltStore, boltStore, snapshots)
		if err != nil {
			return nil, fmt.Errorf("failed to check for existing state: %v", err)
		}

		if !hasState {
			log.Printf("Bootstrapping cluster with node ID %s at %s", s.config.NodeID, transport.LocalAddr())
			configuration := raft.Configuration{
				Servers: []raft.Server{
					{
						ID:      raft.ServerID(s.config.NodeID),
						Address: transport.LocalAddr(),
					},
				},
			}
			if err := s.raft.BootstrapCluster(configuration).Error(); err != nil {
				return nil, fmt.Errorf("could not bootstrap cluster: %w", err)
			}
		}
	} else if s.config.JoinAddr != "" {
		leaderAddr, err := net.ResolveTCPAddr("tcp", s.config.JoinAddr)
		if err != nil {
			return nil, fmt.Errorf("could not resolve address %s to join: %w", s.config.JoinAddr, err)
		}

		// Call add-node API on the leader
		addNodeURL := fmt.Sprintf("http://%s/add-node?followerId=%s&followerAddr=%s",
			leaderAddr.String(), s.config.NodeID, s.config.RaftAdvertiseAddr)

		maxRetries := 30
		for i := range maxRetries {
			log.Printf("Attempting to join cluster via %s (attempt %d/%d)", addNodeURL, i+1, maxRetries)

			resp, err := http.Post(addNodeURL, "application/json", nil)
			if err != nil {
				log.Printf("Failed to call add-node API on leader: %v", err)
			} else {
				defer resp.Body.Close()

				if resp.StatusCode == 200 {
					log.Printf("Successfully joined cluster via %s", addNodeURL)
					break
				}

				body, _ := io.ReadAll(resp.Body)
				log.Printf("Add-node API returned status %d: %s", resp.StatusCode, string(body))
			}

			time.Sleep(2 * time.Second)
			if i == maxRetries-1 {
				return nil, fmt.Errorf("failed to join cluster after %d attempts", maxRetries)
			}
		}
	}

	return s, nil
}

// Apply applies a command to the key-value store via Raft.
func (s *Store) Apply(data []byte) error {
	if s.raft.State() != raft.Leader {
		return fmt.Errorf("not the leader")
	}

	future := s.raft.Apply(data, 5*time.Second)
	if err := future.Error(); err != nil {
		return fmt.Errorf("could not perform apply command via Raft: %w", err)
	}

	if fsmResponse := future.Response(); fsmResponse != nil {
		if fsmErr, ok := fsmResponse.(error); ok {
			return fmt.Errorf("FSM error on apply command: %w", fsmErr)
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

	log.Printf("Handling add follower request for node %s at %s", followerId, followerAddr)
	if err := s.raft.AddVoter(raft.ServerID(followerId), raft.ServerAddress(followerAddr), 0, 0).Error(); err != nil {
		log.Printf("Failed to add voter %s (%s): %s", followerId, followerAddr, err)
		return err
	}
	return nil
}
