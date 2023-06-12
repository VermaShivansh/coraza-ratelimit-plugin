package plugin

import (
	"encoding/json"
	"fmt"
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
	zoneMacro     macro.Macro
	interrupt     types.Interruption
	mutex         *sync.Mutex
}

func (e *Ratelimit) Init(rm rules.RuleMetadata, opts string) error {
	fmt.Println("Ratelimit plugin initiated", opts)
	var err error

	// initiating macro for retrieving name in future
	e.zoneMacro, err = macro.NewMacro(strings.Split(opts, "=")[1])
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	e.Zones = make(map[string]ZoneEvents)

	e.MaxEvents = 200
	e.Window = 1
	e.SweepInterval = 2

	// Generating interrupt - right now static to deny-429
	e.interrupt = types.Interruption{
		RuleID: rm.ID(),
		Action: "deny",
		Status: 429,
	}

	e.mutex = &sync.Mutex{}

	// a service to clean interval
	go func() {
		for {
			time.Sleep(time.Second * time.Duration(e.SweepInterval)) // runs after every SweepInterval duration

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

			fmt.Printf("Cleaned memory for Rule with id %v \n", rm.ID())
		}
	}()

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
	zone_name := e.zoneMacro.Expand(tx)
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
		fmt.Println(e.Zones)
	} else {
		// implement logic after ratelimit exceeded
		fmt.Println("Ratelimit exceeded")
		corazaLogger.Debug().Msg("Ratelimit exceeded")
		tx.Interrupt(&e.interrupt)

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

func prettyPrint(i interface{}) {
	s, _ := json.MarshalIndent(i, "", "\t")
	fmt.Println(string(s))
}

// type lock
var (
	_ rules.Action          = &Ratelimit{}
	_ plugins.ActionFactory = newRatelimit
)
