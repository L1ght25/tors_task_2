package main

import (
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	httpserver "raftdb/internal/http-server"
	"raftdb/internal/raft"

	"github.com/lmittmann/tint"
)

func main() {
	slog.SetDefault(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelInfo,
		}),
	))

	var (
		id       int64
		raftPort int
		httpPort int
		peers    []string
	)

	flag.Int64Var(&id, "id", 0, "Id for replicaset")
	flag.IntVar(&raftPort, "raft_port", 0, "Raft port for replica")
	flag.IntVar(&httpPort, "http_port", 0, "HTTP-port for user requests")
	flag.Parse()
	peers = flag.Args()
	slog.Info("Got another replicas", "peers", peers)

	raftServer := raft.NewRaftServer(id, peers)
	httpServer := httpserver.NewServer(raftServer)

	http.HandleFunc("/put", httpServer.PutHandler)
	http.HandleFunc("/get/{key}", httpServer.GetHandler)
	// http.HandleFunc("/delete", server.DeleteHandler)
	// http.HandleFunc("/cas", server.CompareAndSwapHandler)

	go raftServer.StartRaftServer(fmt.Sprintf(":%d", raftPort))

	slog.Info("HTTP-server starting", "port", httpPort)
	if err := http.ListenAndServe(fmt.Sprintf(":%d", httpPort), nil); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
