package raft

import (
	"context"
	"log/slog"

	"raftdb/internal/proto/pb"
)

func (s *RaftServer) beginElection() {
	s.mu.Lock()
	defer s.mu.Unlock()

	slog.Info("Begin election...", "node", s.id)

	s.currentTerm++
	s.state = CANDIDATE
	s.lastVotedFor = s.id

	votes := 1 // self voted

	s.electionTimer = s.Tick(s.electionTimer, s.electionTimeout, s.beginElection)

	for _, peer := range s.peers {
		go func(peer string) {
			request := &pb.VoteRequest{
				Term:         s.currentTerm,
				CandidateID:  s.id,
				LastLogIndex: int64(len(s.log) - 1),
				LastLogTerm:  int64(s.log[len(s.log)-1].Term),
			}

			response, err := sendRequestVote(peer, request)
			if err == nil && response.VoteGranted {
				s.mu.Lock()
				defer s.mu.Unlock()
				slog.Info("Vote granted!", "node", s.id, "votes", votes)
				votes++
				if votes > len(s.peers)/2 && s.state == CANDIDATE {
					s.becomeLeader()
				}
			}
		}(peer)
	}
}

func (s *RaftServer) becomeLeader() {
	slog.Info("Becoming leader", "node", s.id)

	s.state = LEADER
	s.leaderID = s.id

	go s.sendHeartbeats()

	if s.electionTimer != nil {
		s.electionTimer.Stop()
	}
}

// receiver stuff
func (s *RaftServer) RequestVote(ctx context.Context, req *pb.VoteRequest) (*pb.VoteResponse, error) {
	slog.Info("RequestVote received", "node", s.id, "candidate", req.CandidateID, "request_term", req.Term, "current_term", s.currentTerm)

	if req.Term < s.currentTerm {
		return &pb.VoteResponse{Term: s.currentTerm, VoteGranted: false}, nil
	}

	if s.lastVotedFor == -1 || req.Term > s.currentTerm || s.lastVotedFor == int64(req.CandidateID) {
		lastLogIndex := int64(len(s.log) - 1)
		lastLogTerm := int64(0)
		if lastLogIndex >= 0 {
			lastLogTerm = s.log[lastLogIndex].Term
		}

		// Условие из оригинальной статьи:
		// Кандидат имеет более новый лог, если:
		// - Его `LastLogTerm` больше, чем у нас, или
		// - При равенстве `LastLogTerm` его `LastLogIndex` больше
		if req.LastLogTerm < lastLogTerm || (req.LastLogTerm == lastLogTerm && req.LastLogIndex < lastLogIndex) {
			return &pb.VoteResponse{Term: s.currentTerm, VoteGranted: false}, nil
		}

		s.currentTerm = req.Term
		s.lastVotedFor = req.CandidateID
		s.electionTimer = s.Tick(s.electionTimer, s.electionTimeout, s.beginElection)
		return &pb.VoteResponse{Term: s.currentTerm, VoteGranted: true}, nil
	}

	return &pb.VoteResponse{Term: s.currentTerm, VoteGranted: false}, nil
}
