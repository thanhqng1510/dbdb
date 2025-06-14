package http

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/thanhqng1510/dbdb/store"
)

// Server represents an HTTP server to communicate with store.
type Server struct {
	addr  string
	store store.IStore
}

// NewServer creates a new HTTP server.
func NewServer(addr string, store store.IStore) *Server {
	return &Server{
		addr:  addr,
		store: store,
	}
}

// Start starts the HTTP server. This is a blocking call.
func (s *Server) Start() error {
	log.Printf("Starting HTTP server on %s", s.addr)

	mux := http.NewServeMux()
	mux.HandleFunc("/apply", s.applyHandler)
	mux.HandleFunc("/get", s.getHandler)
	mux.HandleFunc("/add-node", s.addNodeHandler)
	mux.HandleFunc("/remove-node", s.removeNodeHandler)
	return http.ListenAndServe(s.addr, mux)
}

func (s *Server) applyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	defer r.Body.Close()

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Could not read request body for apply operation: %s", err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}

	if err := s.store.Apply(bodyBytes); err != nil {
		log.Printf("Error applying operation: %s", err)
		http.Error(w, fmt.Sprintf("Failed to apply operation: %s", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (s *Server) getHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Only GET method is allowed", http.StatusMethodNotAllowed)
		return
	}

	key := r.URL.Query().Get("key")
	if key == "" {
		http.Error(w, "Key parameter must not be empty", http.StatusBadRequest)
		return
	}

	value, exist := s.store.Get(key)
	if !exist {
		http.Error(w, fmt.Sprintf("Key %s not found", key), http.StatusBadRequest)
		return
	}

	var valStr string
	if value == nil {
		valStr = ""
	} else {
		valStr = value.(string)
	}

	rsp := struct {
		Data string `json:"data"`
	}{valStr}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(rsp); err != nil {
		log.Printf("Could not encode get response: %s", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

func (s *Server) addNodeHandler(w http.ResponseWriter, r *http.Request) {
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

	if err := s.store.AddFollower(followerId, followerAddr); err != nil {
		log.Printf("Failed to add follower: %s", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{err.Error()})
		return
	}

	log.Printf("Successfully added voter %s (%s) to the cluster", followerId, followerAddr)
	w.WriteHeader(http.StatusOK)
}

func (s *Server) removeNodeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method is allowed", http.StatusMethodNotAllowed)
		return
	}

	followerId := r.URL.Query().Get("followerId")

	if followerId == "" {
		http.Error(w, "Missing followerId query parameter", http.StatusBadRequest)
		return
	}

	if err := s.store.RemoveFollower(followerId); err != nil {
		log.Printf("Failed to remove follower: %s", err)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		
		json.NewEncoder(w).Encode(struct {
			Error string `json:"error"`
		}{err.Error()})
		return
	}

	log.Printf("Successfully remove voter %s to the cluster", followerId)
	w.WriteHeader(http.StatusOK)
}
