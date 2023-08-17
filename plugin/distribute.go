package plugin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bytedance/sonic"
	"github.com/redis/go-redis/v9"
	"github.com/vermaShivansh/coraza-ratelimit-plugin/helpers"
)

type Distributed struct {
	Active       bool          // if ratelimit has to be distributed
	SyncInterval time.Duration // syncing with redis at interval
	lastSync     int64         // timestamp for lastSync with redis
	UniqueKey    string        // has to be same for all the instances which have to be synced together
}

var RATELIMIT_KEY = "coraza_ratelimit_key"

// initiate distribution
func (e *Ratelimit) initDistribute(syncInterval time.Duration) error {
	var err error

	unique_key := os.Getenv(RATELIMIT_KEY)

	if unique_key == "" {
		return errors.New("please set a env value with keyname `coraza_ratelimit_key` to use ratelimit distribution ")
	}

	if err = helpers.CheckRatelimitDistributeKey(unique_key); err != nil {
		return err
	}

	e.Distributed.Active = true
	e.Distributed.UniqueKey = unique_key
	e.Distributed.SyncInterval = syncInterval
	e.Distributed.lastSync = 0

	return nil
}

// it uses syncFunc declared below

// syncFunc does the core sync stuff
// it is ran initially to get the initial data from redis
// and runs syncFunc after every syncInterval

func (e *Ratelimit) syncService() {

	//populate initial values of ZoneEvents
	if err := syncFunc(e); err != nil {
		fmt.Println(err)
	}

	ticker := time.NewTicker(time.Second * e.Distributed.SyncInterval)
	for {
		<-ticker.C
		if err := syncFunc(e); err != nil {
			fmt.Println(err)
		}
	}

}

// 1. Obtain lock on redis for unique key (same throughout all the instances of application)
// 2. Fetch ratelimit data
// 3. update the ratelimit data with events of all timestamps>lastSync and timestamp > (currentTime-e.Window) (suppose syncing is done every minute but window is of 5seconds then we should not compute events of timestamp from 0th second - 55th second as they are redundant.)
// 4. Set the updated states in Redis
// 5. Release the lock

func syncFunc(e *Ratelimit) error {
	//read from the redis
	fmt.Println("Syncing with Redis")

	// currentTimeMilli := time.Now().UnixMilli()
	//Obtain Redis lock
	ctx := context.Background()

	redisLock, err := ObtainLock(ctx, e.Distributed.UniqueKey)
	if err != nil {
		return err
	}
	// fmt.Println("LOCK OBTAIN", time.Now().UnixMilli()-currentTimeMilli)

	result, err := GetData(ctx, e.Distributed.UniqueKey)
	if err != nil && err != redis.Nil {
		return fmt.Errorf("error while fetching from redis: %v", err.Error())
	}
	// fmt.Println("GET", time.Now().UnixMilli()-currentTimeMilli)

	currentTime := time.Now().Unix()
	syncedZones := Zones{}

	// if key does not exist-> there is nothing to marshal
	if err != redis.Nil {
		err = sonic.Unmarshal([]byte(result), &syncedZones)
		if err != nil {
			return fmt.Errorf("error while unmarshling result from redis: %v", err.Error())
		}
	}

	//update locally
	e.mutex.Lock()

	for zone_name, zone_events := range syncedZones {
		for timestamp, events_in_timestamp := range zone_events {
			// we dont care about the timestamps which are not in current window
			// we dont care about the timestamps which already have been synced

			if timestamp > e.Distributed.lastSync && timestamp > (currentTime-e.Window) {
				_, ok := e.Zones[zone_name]
				if !ok {
					// create zone in local if doesn't exist and update zone_events in currenttimestamp
					e.Zones[zone_name] = ZoneEvents{
						timestamp: events_in_timestamp,
					}
					break
				} else {
					_, ok := e.Zones[zone_name][timestamp]
					if !ok {
						e.Zones[zone_name][timestamp] = events_in_timestamp
					} else {
						e.Zones[zone_name][timestamp] += events_in_timestamp
					}
				}
			}
		}
	}
	// e.Zones is completely synced now
	syncedZones = e.Zones
	e.mutex.Unlock()

	log.Println("syncedZones", syncedZones)

	jsonStrByteArray, err := sonic.Marshal(syncedZones)
	if err != nil {
		return fmt.Errorf("error encoding JSON: %v", err.Error())
	}

	// set in redis
	err = SetData(ctx, e.Distributed.UniqueKey, string(jsonStrByteArray))
	if err != nil {
		return fmt.Errorf("error in setting value in DB: %v", err.Error())
	}
	// fmt.Println("SET", time.Now().UnixMilli()-currentTimeMilli)

	// Release the lock so other processes or threads can obtain a lock.
	err = redisLock.Release(ctx)
	if err != nil {
		return err
	}

	// fmt.Println("LOCK RELEASE", time.Now().UnixMilli()-currentTimeMilli)

	//updating last sync time with the time when we fetched data from redis
	e.Distributed.lastSync = currentTime

	return nil
}
