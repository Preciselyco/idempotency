package idempotency

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Storage is a interface to implement storing and getting idempotency keys.
// This is what actually implements the state.
type Storage interface {
	Add(ctx context.Context, key string) error
	Get(ctx context.Context, key string) (*RequestStatus, error)
	Complete(ctx context.Context, key string) error
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
func (m *memoryStorage) Add(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storage[key] = &RequestStatus{InProcess: true}

	return nil
}

// Get fetches the RequestStatus for an idempotency key.
func (m *memoryStorage) Get(ctx context.Context, key string) (*RequestStatus, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.storage[key], nil
}

// Complete sets a request to not be in progress, it is then determined to be
// completed and that we should serve the result we got from a previous
// request.
func (m *memoryStorage) Complete(ctx context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.storage[key] = &RequestStatus{InProcess: false}

	return nil
}

type redisStorage struct {
	client    *redis.Client
	expiry    time.Duration
	keyPrefix string
}

// RedisStorageOption is the signature for functional options for the Redis
// storage.
type RedisStorageOption func(*redisStorage)

func WithKeyPrefix(prefix string) RedisStorageOption {
	return func(rs *redisStorage) {
		rs.keyPrefix = prefix
	}
}

// NewMemoryStorage creates a Redis storage for Idempotency-Keys to be able
// to provide a distrigbuted state of the keys.
func NewRedisStorage(client *redis.Client, expiry time.Duration, opts ...RedisStorageOption) *redisStorage {
	s := &redisStorage{
		client:    client,
		expiry:    expiry,
		keyPrefix: "idemp:",
	}

	for _, opt := range opts {
		if opt != nil {
			opt(s)
		}
	}

	return s
}

// Add inserts the initial state of a request with an idempotency key.
func (s *redisStorage) Add(ctx context.Context, key string) error {
	// We use SETNX in order to handle a race condition where the keys can be
	// checked by two processes and find that they do not exist, after which both
	// try to write the key.
	res, err := s.client.SetNX(ctx, s.keyPrefix+key, "in-process", s.expiry).Result()
	if err != nil {
		return fmt.Errorf("failed to set the key %q in redis: %w", key, err)
	}
	if !res {
		return fmt.Errorf("the key %q already exists in redis", key)
	}

	return nil
}

// Get fetches the RequestStatus for an idempotency key.
func (s *redisStorage) Get(ctx context.Context, key string) (*RequestStatus, error) {
	res, err := s.client.Get(ctx, s.keyPrefix+key).Result()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get the key %q from redis: %w", key, err)
	}
	return &RequestStatus{
		InProcess: res == "in-process",
	}, nil
}

// Complete sets a request to not be in progress, it is then determined to be
// completed and that we should serve the result we got from a previous
// request.
func (s *redisStorage) Complete(ctx context.Context, key string) error {
	_, err := s.client.Set(ctx, s.keyPrefix+key, "done", redis.KeepTTL).Result()
	if err != nil {
		return fmt.Errorf("failed to update the key %q in redis: %w", key, err)
	}
	return nil
}
