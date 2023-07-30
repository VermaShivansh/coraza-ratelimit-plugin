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
		Addr: "localhost:6379",
	})

	// lock.
	locker = redislock.New(client)
}

// lockKey is used for both purposes
// 1. Creating a lock with name `lockKey + "key"`
// 2. Key name for a value in Redis
// This way it gives a feel that a lock is associated to a key:value pair
// Same lock cannot be acquired by another instance until one frees it which means same ratelimit events cannot be updated until one instance is done with it
func SetDataWithLock(lockKey string, data string) error {
	ctx := context.Background()
	// Obtain a lock for our given mutex. After this is successful, no one else
	// can obtain the same lock (the same mutex name) until we unlock it.

	// Retry every 100ms, for up-to 3x
	backoff := redislock.LimitRetry(redislock.LinearBackoff(50*time.Millisecond), 10)

	lock, err := locker.Obtain(ctx, lockKey+"lock", 3000*time.Millisecond, &redislock.Options{
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

func SetData(ctx context.Context, key string, value string) error {
	return client.Set(ctx, key, value, 0).Err()
}

func GetData(key string) (string, error) {
	ctx := context.Background()
	return client.Get(ctx, key).Result()
}
