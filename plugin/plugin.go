package plugin

import (
	"encoding/json"
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

type Ratelimit struct {
	Zones         map[string]ZoneEvents
	MaxEvents     int64 // no of requests allowed
	Window        int64 // no of maxEvents in inteval : in seconds
	SweepInterval int64 // cleans memory at interval : in seconds .
	ZoneMacro     macro.Macro
	Action        string
	Status        int // because coraza accepts 'int' in its interrupt struct
	mutex         *sync.Mutex
}

func (e *Ratelimit) Init(rm rules.RuleMetadata, opts string) error {
	log.Println("Initiating Ratelimit plugin with config ", opts)
	var err error

	e.Zones = make(map[string]ZoneEvents)

	//default values
	e.SweepInterval = 5
	e.Action = "drop"
	e.Status = 429

	//parses the configuration and loads values to the struct whilst checking required and valid values
	if err = e.parseConfig(opts); err != nil {
		return fmt.Errorf("Ratelimit config error for ruleID= %v ; errorMsg= %v", rm.ID(), err)
	}

	e.mutex = &sync.Mutex{}

	prettyPrint(e)

	go e.cleanService(rm.ID())

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

	//extract zone
	zone_name := e.ZoneMacro.Expand(tx)
	if zone_name == "" {
		zone_name = "misc" // if in case of empty string or some kind of issue in macro expansion we send all the requests to misc name
	}

	currentTimeInSecond := time.Now().Unix()

	e.mutex.Lock()
	defer e.mutex.Unlock()

	_, ok := e.Zones[zone_name]
	if !ok {
		e.Zones[zone_name] = make(ZoneEvents)
	}

	_, ok = e.Zones[zone_name][currentTimeInSecond]
	if !ok {
		e.Zones[zone_name][currentTimeInSecond] = 0
	}

	// total events for that zone
	var totalEventsOccuredInPreviousWindow int64 = 0
	for i := currentTimeInSecond; i > currentTimeInSecond-e.Window; i-- {
		totalEventsOccuredInPreviousWindow += e.Zones[zone_name][i]
	}

	if totalEventsOccuredInPreviousWindow < e.MaxEvents {
		e.Zones[zone_name][currentTimeInSecond]++
		log.Println(e.Zones)
	} else {
		// implement logic after ratelimit exceeded
		log.Println("Ratelimit exceeded")
		corazaLogger.Debug().Msg("Ratelimit exceeded")
		tx.Interrupt(&types.Interruption{
			RuleID: r.ID(),
			Status: e.Status,
			Action: e.Action,
		})

		return
	}

	prettyPrint(tx.Variables().Rule())

	// get information about current matching SecRule
	// prettyPrint(tx.Collection(variables.MatchedVar).FindAll())
	// prettyPrint(tx.Collection(variables.MatchedVarName).FindAll())
	// prettyPrint(tx.Collection(variables.RemoteAddr).FindAll())
	// prettyPrint(tx.Collection(variables.Rule).FindAll())
	// prettyPrint(tx.Collection(variables.ResponseHeaders).FindAll())

	// tx.Variables().All(func(v variables.RuleVariable, col collection.Collection) bool {
	// 	prettyPrint(map[string]interface{}{"variable": col.Name(), "col": col.FindAll()})
	// 	return true
	// })
}

func (e *Ratelimit) Type() rules.ActionType {
	return rules.ActionTypeNondisruptive
}

func (e *Ratelimit) parseConfig(config string) error {
	// acceptable keys
	var err error

	tokens := strings.Split(config, "&")

	requiredValues := map[string]bool{
		"zone":   false,
		"events": false,
		"window": false,
	}

	for _, token := range tokens {
		pair := strings.Split(token, "=")
		if len(pair) != 2 {
			return fmt.Errorf("invalid usage of = for %v", pair)
		}

		key := pair[0]
		value := pair[1]

		switch key {
		case "zone":
			if e.ZoneMacro, err = macro.NewMacro(value); err != nil {
				return fmt.Errorf("invalid macro name")
			}
			requiredValues[key] = true
		case "events":
			if e.MaxEvents, err = strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("invalid integer value for events")
			}
			requiredValues[key] = true
		case "window":
			if e.Window, err = strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("invalid integer value for window")
			}
			if e.Window == 0 {
				return fmt.Errorf("value 0 is not allowed for key 'window'")
			}
			requiredValues[key] = true
		case "interval":
			if e.SweepInterval, err = strconv.ParseInt(value, 10, 64); err != nil {
				return fmt.Errorf("invalid integer value for interval")
			}
			if e.SweepInterval == 0 {
				return fmt.Errorf("value 0 is not allowed for key 'sweepInterval'")
			}
		case "action":
			if value == "drop" || value == "deny" || value == "redirect" {
				e.Action = value
			} else {
				return fmt.Errorf("action type should be one of 'drop', 'deny', 'redirect'")
			}
		case "status":
			if e.Status, err = strconv.Atoi(value); err != nil {
				return fmt.Errorf("invalid status integer value")
			}
			if e.Status < 0 || e.Status > 500 {
				return fmt.Errorf("status should be in range 0-500")
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
func (e *Ratelimit) cleanService(ruleID int) error {
	ticker := time.NewTicker(time.Second * time.Duration(e.SweepInterval))
	for {
		<-ticker.C
		thresholdTimeStamp := time.Now().Unix() - e.Window
		// aim is to keep events of timestamps greater than threshold timestamp

		fmt.Printf("Removing timestamps less than or equal to %v \n", thresholdTimeStamp)

		e.mutex.Lock()
		for zone_name, zone_timestamps := range e.Zones {
			for timestamp := range zone_timestamps {
				if timestamp <= thresholdTimeStamp {
					delete(e.Zones[zone_name], timestamp)
				} else {
					// breaking out of loop if at any point timestamps start to increase than threshold: it reduces redundant iteration computation
					break
				}
			}
		}
		e.mutex.Unlock()

		fmt.Printf("Cleaned memory for Rule with id %v \n", ruleID)
	}
}

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

// type lock
var (
	_ rules.Action          = &Ratelimit{}
	_ plugins.ActionFactory = newRatelimit
)
