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

func (r *Replicator) ApplyAndReplicate(command string, key string, value, oldValue *string) (bool, error) {
	return r.raftServer.ReplicateLogEntry(command, key, value, oldValue)
}

func (r *Replicator) WaitForRead() error {
	return r.raftServer.WaitForRead()
}
