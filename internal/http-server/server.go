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

func (s *Server) PutHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Command string `json:"command"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := s.replicator.ApplyAndReplicate(req.Command); err != nil {
		http.Error(w, fmt.Sprintf("Failed to apply command, err: %v", err), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) GetHandler(w http.ResponseWriter, r *http.Request) {
	key := r.PathValue("key")
	if key == "" {
		http.Error(w, "Key is required", http.StatusBadRequest)
		return
	}

	if value, ok := db.Get(key); ok {
		w.Write([]byte(fmt.Sprintf("%v", value)))
	} else {
		http.Error(w, "Key not found", http.StatusNotFound)
	}
}

// func (s *Server) DeleteHandler(w http.ResponseWriter, r *http.Request) {
// 	var req struct {
// 		Key string `json:"key"`
// 	}
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request", http.StatusBadRequest)
// 		return
// 	}

// 	command := map[string]string{
// 		"operation": "delete",
// 		"key":       req.Key,
// 	}
// 	if err := s.raft.ApplyCommand(command); err != nil {
// 		http.Error(w, "Failed to apply command", http.StatusInternalServerError)
// 		return
// 	}
// 	w.WriteHeader(http.StatusOK)
// }

// func (s *Server) CompareAndSwapHandler(w http.ResponseWriter, r *http.Request) {
// 	var req struct {
// 		Key      string `json:"key"`
// 		OldValue string `json:"old_value"`
// 		NewValue string `json:"new_value"`
// 	}
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request", http.StatusBadRequest)
// 		return
// 	}

// 	command := map[string]string{
// 		"operation": "cas",
// 		"key":       req.Key,
// 		"old_value": req.OldValue,
// 		"new_value": req.NewValue,
// 	}
// 	if err := s.raft.ApplyCommand(command); err != nil {
// 		http.Error(w, "Failed to apply command", http.StatusInternalServerError)
// 		return
// 	}
// 	w.WriteHeader(http.StatusOK)
// }
