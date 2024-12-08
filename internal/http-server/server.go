package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"raftdb/internal/db"
	"raftdb/internal/raft"
	"raftdb/internal/replicator"
)

type Server struct {
	replicator *replicator.Replicator
}

func NewServer(raft *raft.RaftServer) *Server {
	return &Server{
		replicator: replicator.NewReplicator(raft),
	}
}

func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	if r.URL.Query().Get("consistent") == "true" {
		err := s.replicator.WaitForRead()
		if err != nil {
			http.Error(w, fmt.Errorf("Error while waiting for consistent read: %w", err).Error(), http.StatusInternalServerError)
			return
		}
	}

	if value, ok := db.Get(key); ok {
		w.Write([]byte(fmt.Sprintf("%v", value)))
	} else {
		http.Error(w, "Key not found", http.StatusNotFound)
	}
}

func (s *Server) CreateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("CREATE", req.Key, &req.Value, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply command, err: %v", err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("%v", success)))
}

func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("DELETE", req.Key, nil, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply command, err: %v", err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("%v", success)))
}

func (s *Server) UpdateHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("UPDATE", req.Key, &req.Value, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply command, err: %v", err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("%v", success)))
}

func (s *Server) CompareAndSwapHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Key      string `json:"key"`
		OldValue string `json:"old_value"`
		NewValue string `json:"new_value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	success, err := s.replicator.ApplyAndReplicate("CAS", req.Key, &req.NewValue, &req.OldValue)
	if err != nil {
		http.Error(w, "Failed to apply command", http.StatusInternalServerError)
		return
	}
	w.Write([]byte(fmt.Sprintf("%v", success)))
}
