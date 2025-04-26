package storage

import "sync"

type InMemoryStorage struct {
	mu sync.RWMutex
}
