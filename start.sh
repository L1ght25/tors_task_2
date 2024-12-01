go build ./cmd/main.go
./cmd -http_port=8080 -id=0 -raft_port=5051 localhost:5052 localhost:5053
./cmd -http_port=8081 -id=1 -raft_port=5052 localhost:5051 localhost:5053
./cmd -http_port=8082 -id=2 -raft_port=5053 localhost:5051 localhost:5052