package raft

import (
	"fmt"
	"log/slog"
	"raftdb/internal/proto/pb"
)

func (s *RaftServer) ReplicateLogEntry(command string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.state != LEADER {
		slog.Error("WTF, replicate in replica", "leader", s.leaderID, "node", s.id)
		return fmt.Errorf("cannot handle write on replica. LeaderID: %v", s.leaderID)
	}

	var prevLogIndex int64
	var prevLogTerm int64

	prevLogIndex = int64(len(s.log) - 1)
	if prevLogIndex >= 0 {
		prevLogTerm = s.log[prevLogIndex].Term
	}
	entry := LogEntry{
		Term:    s.currentTerm,
		Command: command,
	}

	s.log = append(s.log, entry)
	slog.Info("master append entry", "leader", s.id, "entry", entry)

	ackCh := make(chan bool, len(s.peers))
	for _, peer := range s.peers {
		go func(peer string) {
			req := &pb.AppendEntriesRequest{
				Term:         s.currentTerm,
				LeaderID:     s.id,
				PrevLogIndex: prevLogIndex,
				PrevLogTerm:  prevLogTerm,
				Entries: []*pb.LogEntry{
					{
						Term:    entry.Term,
						Command: []byte(entry.Command),
					},
				},
				LeaderCommit: s.commitIndex,
			}

			resp, err := sendAppendEntries(peer, req)
			if err == nil && resp.Success {
				ackCh <- true
			} else {
				if err != nil {
					slog.Error("Append new entry error", "leader", s.id, "error", err)
				}

				ackCh <- false
			}
		}(peer)
	}

	ackCount := 1 // self ack
	for i := 0; i < len(s.peers); i++ {
		if <-ackCh {
			ackCount++
		}
		if ackCount > len(s.peers)/2 {
			break
		}
	}

	if ackCount > len(s.peers)/2 {
		slog.Info("Successfully replicated entry, applying...", "leader", s.id, "entry", entry)
		s.applyEntries(int64(len(s.log) - 1))
		return nil
	}

	return fmt.Errorf("cannot replicate entry %+v", entry)
}
