package utils

import (
	"context"
	"errors"
	"time"

	"github.com/NUS-ISS-Agile-Team/ceramicraft-order-mservice/server/repository/dao/redis"
	goredis "github.com/redis/go-redis/v9"
)

// Locker defines the interface for distributed lock operations
type Locker interface {
	// Lock attempts to acquire the distributed lock
	Lock(ctx context.Context) error

	// TryLock attempts to acquire the lock with retry logic
	TryLock(ctx context.Context, maxRetries int, retryDelay time.Duration) error

	// Unlock releases the distributed lock
	Unlock(ctx context.Context) error

	// Refresh extends the lock expiration time
	Refresh(ctx context.Context) error

	// IsLocked checks if the lock is currently held (by any instance)
	IsLocked(ctx context.Context) (bool, error)

	// IsLockedByMe checks if the lock is held by this specific instance
	IsLockedByMe(ctx context.Context) (bool, error)
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

// TryLock attempts to acquire the lock with retry logic
// maxRetries: maximum number of retry attempts
// retryDelay: delay between retry attempts
func (l *DistributedLock) TryLock(ctx context.Context, maxRetries int, retryDelay time.Duration) error {
	for i := 0; i < maxRetries; i++ {
		err := l.Lock(ctx)
		if err == nil {
			return nil
		}

		if err != ErrLockFailed {
			return err
		}

		// Wait before retrying
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			continue
		}
	}

	return ErrLockFailed
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

// Refresh extends the lock expiration time
// Useful for long-running operations that need to keep the lock
func (l *DistributedLock) Refresh(ctx context.Context) error {
	// Use Lua script to atomically check ownership and extend expiration
	refreshScript := `
if redis.call("get", KEYS[1]) == ARGV[1] then
    return redis.call("expire", KEYS[1], ARGV[2])
else
    return 0
end
`
	script := goredis.NewScript(refreshScript)
	result, err := script.Run(ctx, redis.RedisClient, []string{l.key}, l.value, int(l.expiration.Seconds())).Result()
	if err != nil {
		return err
	}

	refreshed, ok := result.(int64)
	if !ok || refreshed == 0 {
		return ErrLockNotHeld
	}

	return nil
}

// IsLocked checks if the lock is currently held (by any instance)
func (l *DistributedLock) IsLocked(ctx context.Context) (bool, error) {
	result, err := redis.RedisClient.Exists(ctx, l.key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// IsLockedByMe checks if the lock is held by this specific instance
func (l *DistributedLock) IsLockedByMe(ctx context.Context) (bool, error) {
	value, err := redis.RedisClient.Get(ctx, l.key).Result()
	if err == goredis.Nil {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return value == l.value, nil
}
