package store

import (
	"encoding/json"
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
	NodeID    string
	RaftDir   string
	RaftAddr  string
	HttpAddr  string
	Bootstrap bool
}

// Store manages the Raft consensus and the key-value data.
type Store struct {
	nodeID   string
	raftDir  string
	raftAddr string
	httpAddr string
	raft     *raft.Raft
	data     *sync.Map
}

// NewStore creates and initializes a new Store.
func NewStore(cfg Config) (*Store, error) {
	s := &Store{
		nodeID:   cfg.NodeID,
		raftDir:  cfg.RaftDir,
		raftAddr: cfg.RaftAddr,
		httpAddr: cfg.HttpAddr,
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
	transport, err := raft.NewTCPTransport(s.raftAddr, tcpAddr, 10, time.Second*10, os.Stderr)
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

// StartHttpServer starts the HTTP server for the store.
func (s *Store) StartHttpServer() error {
	log.Printf("Starting HTTP server on %s", s.httpAddr)

	mux := http.NewServeMux()
	mux.HandleFunc("/set", s.setHandler)
	mux.HandleFunc("/get", s.getHandler)
	mux.HandleFunc("/join", s.joinHandler)
	return http.ListenAndServe(s.httpAddr, mux)
}

func (s *Store) setHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost { // Added method check for robustness
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Could not read request body for set: %s", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	future := s.raft.Apply(bodyBytes, 500*time.Millisecond)
	if err := future.Error(); err != nil {
		log.Printf("Could not apply set command via Raft: %s", err)
		http.Error(w, "Failed to set value (Raft error)", http.StatusInternalServerError)
		return
	}

	if fsmResponse := future.Response(); fsmResponse != nil {
		if fsmErr, ok := fsmResponse.(error); ok {
			log.Printf("FSM error on set command: %s", fsmErr)
			http.Error(w, fmt.Sprintf("Failed to set value (FSM error: %s)", fsmErr), http.StatusInternalServerError)
			return
		}
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Store) getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet { // Added method check
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")

	value, _ := s.data.Load(key)
	var valStr string
	if value == nil {
		valStr = ""
	} else {
		valStr = value.(string) // Assuming all stored values are strings
	}

	rsp := struct { Data string `json:"data"` }{valStr}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rsp); err != nil {
		log.Printf("Could not encode get response: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Store) joinHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	followerId := r.URL.Query().Get("followerId")
	followerAddr := r.URL.Query().Get("followerAddr")

	if followerId == "" || followerAddr == "" {
		http.Error(w, "Missing followerId or followerAddr query parameters", http.StatusBadRequest)
		return
	}

	if s.raft.State() != raft.Leader {
		log.Printf("Join request for %s denied: this node (%s) is not the leader (state: %s)", followerId, s.nodeID, s.raft.State())
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(struct{ Error string `json:"error"` }{"Not the leader"})
		return
	}

	log.Printf("Handling join request for node %s at %s", followerId, followerAddr)
	if err := s.raft.AddVoter(raft.ServerID(followerId), raft.ServerAddress(followerAddr), 0, 0).Error(); err != nil {
		log.Printf("Failed to add voter %s (%s): %s", followerId, followerAddr, err)
		http.Error(w, fmt.Sprintf("Failed to add follower: %s", err), http.StatusBadRequest) // As per original
		return
	}
	log.Printf("Successfully added voter %s (%s) to the cluster", followerId, followerAddr)
	w.WriteHeader(http.StatusOK)
}