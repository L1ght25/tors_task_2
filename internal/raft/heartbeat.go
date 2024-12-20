package raft

import (
	"log/slog"
	"raftdb/internal/db"
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
			for {
				s.mu.Lock()
				nextIndex, exists := s.nextIndex[peer]
				if !exists {
					s.nextIndex[peer] = 0
					nextIndex = 0
				}
				entries := s.log[nextIndex:]
				entriesProto := make([]*pb.LogEntry, len(entries))
				for i, entry := range entries {
					entriesProto[i] = &pb.LogEntry{
						Term:     entry.Term,
						Command:  entry.Command,
						Key:      entry.Key,
						Value:    entry.Value,
						OldValue: entry.OldValue,
					}
				}

				var PrevLogTerm int64
				if nextIndex > 0 {
					PrevLogTerm = s.log[nextIndex-1].Term
				}

				req := &pb.AppendEntriesRequest{
					Term:         s.currentTerm,
					LeaderID:     s.id,
					LeaderCommit: s.commitIndex,
					PrevLogIndex: nextIndex - 1,
					PrevLogTerm:  PrevLogTerm,
					Entries:      entriesProto,
				}
				s.mu.Unlock()

				resp, err := sendAppendEntries(peer, req)
				if err != nil {
					slog.Error("heartbeat from leader error", "error", err, "leader", s.id, "node", peer)
					return
				}

				if nextIndex == 0 {
					break
				}

				s.mu.Lock()
				if resp.Success {
					s.nextIndex[peer] = nextIndex + int64(len(entries))

					// check if we need to commit entry from log
					count := 0
					for _, nextInd := range s.nextIndex {
						if s.commitIndex+1 < nextInd {
							count++
						}
					}
					if count > len(s.peers)/2 {
						slog.Info("There is uncommited entry in log, commiting...", "leader", s.id, "index", s.commitIndex)
						s.commitIndex++
						entry := s.log[s.commitIndex]
						db.ProcessWrite(entry.Command, entry.Key, entry.Value, entry.OldValue)
					}

					s.mu.Unlock()
					break
				}
				s.nextIndex[peer] = nextIndex - 1
				slog.Info("Replica not in sync! Decrementing next index and retrying", "leader", s.id, "node", peer)
				s.mu.Unlock()
			}
		}(peer)
	}

	s.heartbeatTimer = s.Tick(s.heartbeatTimer, s.heartbeatTimeout, s.sendHeartbeats)
}
