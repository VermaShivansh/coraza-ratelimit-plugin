package plugin

import (
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/corazawaf/coraza/v3/debuglog"
	"github.com/corazawaf/coraza/v3/experimental/plugins"
	"github.com/corazawaf/coraza/v3/macro"
	"github.com/corazawaf/coraza/v3/rules"
	"github.com/corazawaf/coraza/v3/types"
)

func init() {
	// Register the plugin
	plugins.RegisterAction("ratelimit", newRatelimit)
}

func newRatelimit() rules.Action {
	return &Ratelimit{}
}

type ZoneEvents map[int64]int64 // unixTimestamp in seconds containing requests per second

type Zones map[string]ZoneEvents

type Ratelimit struct {
	Zones         Zones
	MaxEvents     int64         // no of requests allowed
	Window        int64         // no of maxEvents in inteval : in seconds
	SweepInterval time.Duration // cleans memory at interval : in seconds .
	ZoneMacros    []macro.Macro
	Action        string
	Status        int // because coraza accepts 'int' in its interrupt struct
	mutex         *sync.Mutex
	Distributed   Distributed
}

func (e *Ratelimit) Init(rm rules.RuleMetadata, opts string) error {
	log.Printf("Initiating Ratelimit plugin with config for ruleID= %v and opts= %v", rm.ID(), opts)
	var err error

	e.Zones = make(map[string]ZoneEvents)

	//default values
	e.SweepInterval = time.Duration(5) //5 seconds
	e.Action = "drop"
	e.Status = 429

	//parses the configuration and loads values to the struct whilst checking required and valid values
	if err = e.parseConfig(opts); err != nil {
		return fmt.Errorf("Ratelimit config error for ruleID= %v ; errorMsg= %v", rm.ID(), err)
	}

	e.mutex = &sync.Mutex{}

	go e.memoryOptimizingService(rm.ID())

	if e.Distributed.Active {
		go e.syncService()
	}

	return nil
}

// right now coraza logger are not used properly - please don't judge :)
// Print statements will be removed before finalizing

func (e *Ratelimit) Evaluate(r rules.RuleMetadata, tx rules.TransactionState) {
	corazaLogger := tx.DebugLogger().With(
		debuglog.Str("action", "ratelimit"),
		debuglog.Int("rule_id", r.ID()),
	)

	corazaLogger.Debug().Msg("Evaluating ratelimit plugin")

	var request_allowed bool

	//extract zone

	// MultiZones behave in 'OR' manner, REQUEST will be allowed if any one of the zone is allowing the request

	currentTimeInSecond := time.Now().Unix()

	for _, zoneMacro := range e.ZoneMacros {
		zone_name := zoneMacro.Expand(tx)
		if zone_name == "" {
			zone_name = "misc" // if in case of empty string or some kind of issue in macro expansion we send all the requests to misc name
		}
		var totalEventsOccuredInPreviousWindow int64 = 0

		e.mutex.Lock()

		_, ok := e.Zones[zone_name]
		if !ok {
			e.Zones[zone_name] = make(ZoneEvents)
		}

		_, ok = e.Zones[zone_name][currentTimeInSecond]
		if !ok {
			e.Zones[zone_name][currentTimeInSecond] = 0
		}

		// total events for that zone
		for i := currentTimeInSecond; i > currentTimeInSecond-e.Window; i-- {
			totalEventsOccuredInPreviousWindow += e.Zones[zone_name][i]
		}

		if totalEventsOccuredInPreviousWindow < e.MaxEvents {
			e.Zones[zone_name][currentTimeInSecond]++
			request_allowed = true // we could have use return here but this is done for following reason
			// suppose 10rps are allowed and 10 requests came with 2 macros %{REQUEST_HEADERS.host} %{REQUEST_HEADERS.authority}
			// first 10 request have host value as 'localhost:3000' and authority as 'abc'
			//
			// 2 CASES: If the 11th request is received within the same second and
			// 1st CASE: has host value as 'localhost:3000' and authority as 'abc', request won't be allowed as 10 requests for both the values of authority and host have exhausted.
			// 2nd CASE: has host value as 'localhost:3000' but authority as 'xyz', request will be allowed as 10 requests for host has been fulfilled but a new value of authority has be received.
		}
		// else {
		// log.Printf("Request denied on basis of %v", zoneMacro.String())
		// }

		e.mutex.Unlock()
	}
	// log.Println(e.Zones)

	if request_allowed {
		return
	}

	// implement logic after ratelimit exceeded
	corazaLogger.Debug().Msg("Ratelimit exceeded")
	tx.Interrupt(&types.Interruption{
		RuleID: r.ID(),
		Status: e.Status,
		Action: e.Action,
	})
}

func (e *Ratelimit) Type() rules.ActionType {
	return rules.ActionTypeNondisruptive
}

func (e *Ratelimit) parseConfig(config string) error {
	// acceptable keys
	var err error

	tokens := strings.Split(config, "&")

	requiredValues := map[string]bool{
		"zone[]": false,
		"events": false,
		"window": false,
	}

	for _, token := range tokens {
		key, value, found := strings.Cut(token, "=")
		if !found || key == "" || value == "" || strings.Contains(value, "=") {
			return fmt.Errorf("invalid usage of = for %v", token)
		}

		switch key {
		case "zone[]":
			var ZoneMacro macro.Macro
			if ZoneMacro, err = macro.NewMacro(value); err != nil {
				return fmt.Errorf("invalid macro name: %v", value)
			}
			e.ZoneMacros = append(e.ZoneMacros, ZoneMacro)
			requiredValues[key] = true
		case "events":
			if e.MaxEvents, err = strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("invalid integer value for events: %v", value)
			}
			requiredValues[key] = true
		case "window":
			if e.Window, err = strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("invalid integer value for window: %v", value)
			}
			if e.Window == 0 {
				return errors.New("value 0 is not allowed for key 'window'")
			}
			requiredValues[key] = true
		case "interval":
			var interval int
			if interval, err = strconv.Atoi(value); err != nil {
				return fmt.Errorf("invalid integer value for interval: %v", value)
			}
			if interval == 0 {
				return errors.New("value 0 is not allowed for key 'sweepInterval'")
			}
			e.SweepInterval = time.Duration(interval)
		case "action":
			if value == "drop" || value == "deny" || value == "redirect" {
				e.Action = value
			} else {
				return errors.New("action type should be one of 'drop', 'deny', 'redirect'")
			}
		case "status":
			if e.Status, err = strconv.Atoi(value); err != nil {
				return fmt.Errorf("invalid status integer value: %v", value)
			}
			if e.Status < 0 || e.Status > 500 {
				return fmt.Errorf("status should be in range 0-500: received: %v ", value)
			}
		case "distribute_interval":
			var interval int
			if interval, err = strconv.Atoi(value); err != nil {
				return fmt.Errorf("invalid distribute_interval integer value: %v", value)
			}
			//initiate distribution
			if err = e.initDistribute(time.Duration(interval)); err != nil {
				return fmt.Errorf("error in initiating distribution: %v", err.Error())
			}
		default:
			return fmt.Errorf("%v is not allowed", key)
		}
	}

	// check for required values
	for key, found := range requiredValues {
		if !found {
			return fmt.Errorf("'%v' is required", key)
		}
	}

	return nil
}

// a service to clean interval
func (e *Ratelimit) memoryOptimizingService(ruleID int) {
	ticker := time.NewTicker(time.Second * e.SweepInterval)
	for {
		<-ticker.C
		thresholdTimeStamp := time.Now().Unix() - e.Window
		// aim is to keep events of timestamps greater than threshold timestamp

		// fmt.Printf("Removing timestamps less than or equal to %v \n", thresholdTimeStamp)
		e.mutex.Lock()
		for zone_name, zone_timestamps := range e.Zones {
			for timestamp := range zone_timestamps {
				if timestamp <= thresholdTimeStamp {
					delete(e.Zones[zone_name], timestamp)
				}
			}
			//removes zones with no timestamps
			if len(zone_timestamps) == 0 {
				delete(e.Zones, zone_name)
			}
		}
		e.mutex.Unlock()

		// log.Printf("Cleaned memory for Rule with id %v \n", ruleID)
	}
}
