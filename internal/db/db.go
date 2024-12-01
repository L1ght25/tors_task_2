package db

import (
	"fmt"
	"log/slog"
	"sync"
)

var kv sync.Map

func Get(key any) (any, bool) {
	return kv.Load(key)
}

func ProcessWrite(command, key string, value, oldValue *string) (bool, error) {
	if len(command) == 0 {
		slog.Error("Invalid command", "command", command)
		return false, fmt.Errorf("Empty command")
	}

	switch command {
	case "CREATE":
		if value == nil {
			slog.Error("Null value", "key", key)
			return false, fmt.Errorf("Null value")
		}
		if _, exists := kv.Load(key); exists {
			return false, fmt.Errorf("already exists")
		}
		kv.Store(key, *value)
	case "DELETE":
		_, existed := kv.LoadAndDelete(key)
		return existed, nil
	case "UPDATE":
		if value == nil {
			slog.Error("Null value", "key", key)
		}
		_, exists := kv.Load(key)
		if !exists {
			return false, nil
		}
		kv.Store(key, *value)
	case "CAS":
		if value == nil {
			slog.Error("Null new value", "key", key)
			return false, fmt.Errorf("Null new value")
		}
		if oldValue == nil {
			slog.Error("Null old value", "key", key)
			return false, fmt.Errorf("Null old value")
		}
		return kv.CompareAndSwap(key, *oldValue, *value), nil
	default:
		slog.Error("Invalid command: expected CREATE/DELETE/UPDATE/CAS", "command", command)
		return false, fmt.Errorf("Invalid command: expected CREATE/DELETE/UPDATE/CAS, got %v", command)
	}

	return true, nil
}
