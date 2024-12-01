package db

import (
	"log/slog"
	"strings"
	"sync"
)

var kv sync.Map

func Get(key any) (any, bool) {
	return kv.Load(key)
}

func isValidCommand(parsedCommand []string, size int) bool {
	if len(parsedCommand) != size {
		slog.Error("Invalid command", "command", parsedCommand)
		return false
	}
	return true
}

func Put(command string) bool {
	parsedCommand := strings.Split(command, " ")
	if len(command) == 0 || len(parsedCommand) == 0 {
		slog.Error("Invalid command", "command", command)
		return false
	}

	switch parsedCommand[0] {
	case "CREATE":
		if !isValidCommand(parsedCommand, 3) {
			break
		}
		key := parsedCommand[1]
		value := parsedCommand[2]
		kv.Store(key, value)
	case "DELETE":
		if !isValidCommand(parsedCommand, 2) {
			break
		}
		key := parsedCommand[1]
		kv.Delete(key)
	case "UPDATE":
		if !isValidCommand(parsedCommand, 3) {
			break
		}
		key := parsedCommand[1]
		value := parsedCommand[2]
		kv.Store(key, value)
	case "CAS":
		if !isValidCommand(parsedCommand, 4) {
			break
		}
		key := parsedCommand[1]
		oldValue := parsedCommand[2]
		newValue := parsedCommand[3]
		return kv.CompareAndSwap(key, oldValue, newValue)
	default:
		slog.Error("Invalid command: expected CREATE/DELETE/UPDATE/CAS", "command", command)
	}

	return true
}
