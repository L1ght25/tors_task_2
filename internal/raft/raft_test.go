package raft_test

import (
	"fmt"
	"log"
	"log/slog"
	"net"
	"os"
	httpserver "raftdb/internal/http-server"
	"raftdb/internal/proto/pb"
	"raftdb/internal/raft"
	"testing"
	"time"

	"github.com/lmittmann/tint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc"
)

const (
	BeginTestRaftPort = 5050
	BeginTestHttpPort = 8080
)

type TestRaftServer struct {
	raftServer *raft.RaftServer
	grpcServer *grpc.Server // for stops
	raftPort   string

	httpServer *httpserver.Server
	httpPort   string
}

func NewTestServer(id int64, peers []string, raftPort, httpPort int64) *TestRaftServer {
	raftServer := raft.NewRaftServer(id, peers)
	httpServer := httpserver.NewServer(raftServer)

	return &TestRaftServer{
		raftServer: raftServer,
		httpServer: httpServer,

		raftPort: fmt.Sprintf(":%d", raftPort),
		httpPort: fmt.Sprintf(":%d", httpPort),
	}
}

func StartTestServer(t *TestRaftServer) {
	grpcServer := grpc.NewServer()
	pb.RegisterRaftServer(grpcServer, t.raftServer)
	t.grpcServer = grpcServer

	t.raftServer.StartTimeouts()

	lis, err := net.Listen("tcp", t.raftPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	go func() {
		slog.Info("Raft-server starting", "port", t.raftPort)
		if err := t.grpcServer.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()
}

func StopTestServer(t *TestRaftServer) {
	t.grpcServer.GracefulStop()
	t.raftServer.ResetTimeouts()
}

func Filter(ss []string, test func(string) bool) (ret []string) {
	for _, s := range ss {
		if test(s) {
			ret = append(ret, s)
		}
	}
	return
}

func NewTestCluster(count int) []*TestRaftServer {
	result := make([]*TestRaftServer, 0, count)

	peers := []string{}
	for id := 0; id < count; id++ {
		peers = append(peers, fmt.Sprintf("localhost:%d", BeginTestRaftPort+id))
	}

	for id := int64(0); id < int64(count); id++ {
		newServer := NewTestServer(
			id,
			Filter(peers, func(s string) bool {
				return s != fmt.Sprintf("localhost:%d", BeginTestRaftPort+id)
			}),
			BeginTestRaftPort+id,
			BeginTestHttpPort+id,
		)

		StartTestServer(newServer)
		result = append(result, newServer)
	}

	return result
}

func TestLeaderElection(t *testing.T) {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelInfo,
		}),
	))

	cluster := NewTestCluster(5)

	assert.Equal(t, int64(-1), cluster[0].raftServer.GetLeaderID())

	time.Sleep(10 * time.Second)
	assert.Equal(t, int64(0), cluster[0].raftServer.GetLeaderID())

	StopTestServer(cluster[0])
	time.Sleep(15 * time.Second)
	assert.Greater(t, cluster[1].raftServer.GetLeaderID(), int64(0))
	assert.Less(t, cluster[1].raftServer.GetLeaderID(), int64(5))
}

func TestLogReplication(t *testing.T) {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelInfo,
		}),
	))

	cluster := NewTestCluster(5)
	time.Sleep(10 * time.Second)

	value := "2"
	success, err := cluster[0].raftServer.ReplicateLogEntry("CREATE", "1", &value, nil)
	assert.NoError(t, err)
	assert.Equal(t, true, success)

	time.Sleep(5 * time.Second)

	for id := 1; id < 5; id++ {
		assert.Equal(t, 2, cluster[id].raftServer.LogLength()) // ["init", "create"]
	}
}

func TestLogSync(t *testing.T) {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelInfo,
		}),
	))

	cluster := NewTestCluster(5)
	time.Sleep(10 * time.Second)
	StopTestServer(cluster[1])

	value := "2"
	success, err := cluster[0].raftServer.ReplicateLogEntry("CREATE", "1", &value, nil)
	assert.NoError(t, err)
	assert.Equal(t, true, success)
	success, err = cluster[0].raftServer.ReplicateLogEntry("CREATE", "2", &value, nil)
	assert.NoError(t, err)
	assert.Equal(t, true, success)
	success, err = cluster[0].raftServer.ReplicateLogEntry("CREATE", "3", &value, nil)
	assert.NoError(t, err)
	assert.Equal(t, true, success)

	time.Sleep(5 * time.Second)

	for id := 2; id < 5; id++ {
		assert.Equal(t, 4, cluster[id].raftServer.LogLength())
	}

	StartTestServer(cluster[1])
	time.Sleep(5 * time.Second)
	assert.Equal(t, 4, cluster[1].raftServer.LogLength())
}
