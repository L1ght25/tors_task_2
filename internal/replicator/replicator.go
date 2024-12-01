package replicator

import "raftdb/internal/raft"

// dispatcher for raft magic
type Replicator struct {
	raftServer *raft.RaftServer
}

func NewReplicator(raftServer *raft.RaftServer) *Replicator {
	return &Replicator{
		raftServer: raftServer,
	}
}

func (r *Replicator) ApplyAndReplicate(command string) error {
	return r.raftServer.ReplicateLogEntry(command)
}
