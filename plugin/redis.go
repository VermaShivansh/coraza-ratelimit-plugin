package plugin

import (
	"fmt"
	"log"
	"time"

	"github.com/bsm/redislock"
	"github.com/redis/go-redis/v9"
	"golang.org/x/net/context"
)

type RedisHelpers interface {
	SetDataWithLock(lockKey string, data string) error
	SetData(key string, data string) error
	GetData(key string) (string, error)
}

var client *redis.Client
var locker *redislock.Client

func init() {
	// Initialize the Redis client
	client = redis.NewClient(&redis.Options{
		Addr:     "13.39.121.216:6380",
		Protocol: 1,
	})

	// lock.
	locker = redislock.New(client)
}

func SetData(ctx context.Context, key string, value string) error {
	return client.Set(ctx, key, value, 0).Err()
}

func GetData(ctx context.Context, key string) (string, error) {
	return client.Get(ctx, key).Result()
}

func ObtainLock(ctx context.Context, lockKey string) (*redislock.Lock, error) {
	// retry 50 times every 60ms to obtain lock basically 3 second
	backoff := redislock.LimitRetry(redislock.LinearBackoff(60*time.Millisecond), 50)

	// lock is released automatically after 1500millisecond
	lock, err := locker.Obtain(ctx, "lock_"+lockKey, 1500*time.Millisecond, &redislock.Options{
		RetryStrategy: backoff,
	})
	if err == redislock.ErrNotObtained {
		fmt.Println("Could not obtain lock!")
		return nil, err
	} else if err != nil {
		fmt.Println(err)
		return nil, err
	}

	return lock, nil
}

// lockKey is used for both purposes
// 1. Creating a lock with name `"lock_"+lockKey`
// 2. Key name for a value in Redis
// This way it gives a feel that a lock is associated to a key:value pair
// Same lock cannot be acquired by another instance until one frees it which means same ratelimit events cannot be updated until one instance is done with it
func SetDataWithLock(lockKey string, data string) error {
	ctx := context.Background()
	// Obtain a lock for our given mutex. After this is successful, no one else
	// can obtain the same lock (the same mutex name) until we unlock it.

	// Retry every 100ms, for up-to 3x
	backoff := redislock.LimitRetry(redislock.LinearBackoff(50*time.Millisecond), 10)

	lock, err := locker.Obtain(ctx, "lock"+lockKey, 3000*time.Millisecond, &redislock.Options{
		RetryStrategy: backoff,
	})
	if err == redislock.ErrNotObtained {
		fmt.Println("Could not obtain lock!")
		return err
	} else if err != nil {
		log.Fatalln(err)
		return err
	}

	// Do your work that requires the lock.
	if err := SetData(ctx, lockKey, data); err != nil {
		fmt.Println("error", err)
		return err
	}

	// Release the lock so other processes or threads can obtain a lock.
	err = lock.Release(ctx)
	if err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}
