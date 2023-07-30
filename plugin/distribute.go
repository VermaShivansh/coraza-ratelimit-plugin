package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

type Distrubute struct {
	Active       bool
	SyncInterval time.Duration
	lastSync     int64
}

func (e *Ratelimit) syncService() {
	ticker := time.NewTicker(e.Distributed.SyncInterval)

	syncFunc := func() error {
		//read from the redis
		fmt.Println("Syncing with Redis")

		// currentTimeMilli := time.Now().UnixMilli()
		//Obtain Redis lock
		ctx := context.Background()

		redisLock, err := ObtainLock(ctx, e.UniqueKey)
		if err != nil {
			return err
		}
		// fmt.Println("LOCK OBTAIN", time.Now().UnixMilli()-currentTimeMilli)

		result, err := GetData(ctx, e.UniqueKey)
		if err != nil && err != redis.Nil {
			return fmt.Errorf("error while fetching from redis: %v", err.Error())
		}
		// fmt.Println("GET", time.Now().UnixMilli()-currentTimeMilli)

		currentTime := time.Now().Unix()
		syncedZones := Zones{}

		// if key does not exist-> there is nothing to marshal
		if err != redis.Nil {
			err = json.Unmarshal([]byte(result), &syncedZones)
			if err != nil {
				return fmt.Errorf("error while unmarshling result from redis: %v", err.Error())
			}
		}

		//update locally
		e.mutex.Lock()

		for zone_name, zone_events := range syncedZones {
			for timestamp, events_in_timestamp := range zone_events {
				// we dont care about the timestamps whose window is already passed
				// we dont care about the timestamps which already have been synced

				if timestamp > e.Distributed.lastSync && timestamp > (currentTime-e.Window) {
					_, ok := e.Zones[zone_name]
					if !ok {
						// create macro for it and update zone_events in currenttimestamp
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

		jsonStrByteArray, err := json.Marshal(syncedZones)
		if err != nil {
			return fmt.Errorf("error encoding JSON: %v", err.Error())
		}

		// set in redis
		err = SetData(ctx, e.UniqueKey, string(jsonStrByteArray))
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

	//populate initial values of ZoneEvents
	if err := syncFunc(); err != nil {
		fmt.Println(err)
	}

	for {
		<-ticker.C
		if err := syncFunc(); err != nil {
			fmt.Println(err)
		}
	}

}
