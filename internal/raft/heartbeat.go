package raft

import (
	"log/slog"
	"raftdb/internal/proto/pb"
)

func (s *RaftServer) sendHeartbeats() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != LEADER {
		return
	}

	for _, peer := range s.peers {
		go func(peer string) {
			req := &pb.AppendEntriesRequest{
				Term:         s.currentTerm,
				LeaderID:     s.id,
				LeaderCommit: s.commitIndex,
			}
			_, err := sendAppendEntries(peer, req)
			if err != nil {
				slog.Error("heartbeat from leader error", "error", err, "leaderID", s.id, "node", peer)
			}
		}(peer)
	}

	s.heartbeatTimer = s.Tick(s.heartbeatTimer, s.heartbeatTimeout, s.sendHeartbeats)
}
