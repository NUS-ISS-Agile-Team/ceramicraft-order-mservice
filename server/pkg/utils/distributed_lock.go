package utils

import (
	"context"
	"errors"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/dao/redis"
	goredis "github.com/redis/go-redis/v9"
)

// mockgen -source=./distributed_lock.go -destination=./mocks/distributed_lock_mock.go -package=mocks

// Locker defines the interface for distributed lock operations
type Locker interface {
	// Lock attempts to acquire the distributed lock
	Lock(ctx context.Context) error

	// Unlock releases the distributed lock
	Unlock(ctx context.Context) error
}

// DistributedLock represents a Redis-based distributed lock
type DistributedLock struct {
	key        string
	value      string
	expiration time.Duration
}

var (
	MyDistributedLock *DistributedLock
	ErrLockFailed     = errors.New("failed to acquire lock")
	ErrUnlockFailed   = errors.New("failed to release lock")
	ErrLockNotHeld    = errors.New("lock not held by this instance")
)

// Lua script for atomic unlock (only unlock if the lock is held by this instance)
const unlockScript = `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("del", KEYS[1])
else
    return 0
end
`

// GetDistributedLock creates a new distributed lock instance
// key: the lock key in Redis
// value: unique identifier for this lock holder (e.g., UUID or instance ID)
// expiration: lock TTL to prevent deadlock if holder crashes
func GetDistributedLock(key string, value string, expiration time.Duration) *DistributedLock {
	MyDistributedLock = &DistributedLock{
		key:        key,
		value:      value,
		expiration: expiration,
	}
	return MyDistributedLock
}

// Lock attempts to acquire the distributed lock using SET NX EX
// Returns nil on success, ErrLockFailed if lock is already held by another instance
func (l *DistributedLock) Lock(ctx context.Context) error {
	// SET key value NX EX expiration
	// NX: only set if key does not exist
	// EX: set expiration time in seconds
	result, err := redis.RedisClient.SetNX(ctx, l.key, l.value, l.expiration).Result()
	if err != nil {
		return err
	}

	if !result {
		return ErrLockFailed
	}

	return nil
}

// Unlock releases the distributed lock using Lua script
// Only succeeds if the lock is held by this instance (value matches)
func (l *DistributedLock) Unlock(ctx context.Context) error {
	script := goredis.NewScript(unlockScript)
	result, err := script.Run(ctx, redis.RedisClient, []string{l.key}, l.value).Result()
	if err != nil {
		return err
	}

	// result is 1 if deleted, 0 if key not found or value mismatch
	deleted, ok := result.(int64)
	if !ok || deleted == 0 {
		return ErrLockNotHeld
	}

	return nil
}