package idempotency

import "sync"

// Storage is a interface to implement storing and getting idempotency keys.
// This is what actually implements the state.
type Storage interface {
	Add(key string) error
	Get(key string) (*RequestStatus, error)
	Complete(key string) error
}

type memoryStorage struct {
	storage map[string]*RequestStatus
	mu      sync.RWMutex
}

// NewMemoryStorage creates a memory storage for Idempotency-Keys to be able
// to provide stateful functionality.
func NewMemoryStorage() *memoryStorage {
	return &memoryStorage{
		storage: make(map[string]*RequestStatus),
	}
}

// Add inserts the initial state of a request with an idempotency key.
func (m *memoryStorage) Add(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storage[key] = &RequestStatus{InProcess: true}

	return nil
}

// Get fetches the RequestStatus for an idempotency key.
func (m *memoryStorage) Get(key string) (*RequestStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.storage[key], nil
}

// Complete sets a request to not be in progress, it is then determined to be
// completed and that we should serve the result we got from a previous
// request.
func (m *memoryStorage) Complete(key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storage[key] = &RequestStatus{InProcess: false}

	return nil
}
