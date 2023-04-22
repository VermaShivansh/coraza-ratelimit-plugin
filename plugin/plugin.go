package plugin

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/corazawaf/coraza/v3/debuglog"
	"github.com/corazawaf/coraza/v3/experimental/plugins"
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

type Ratelimit struct {
	allowedCount           int
	remainingCount         int
	clearAfterSeconds      int
	lastClearRatelimitTime time.Time
	interrupt              *types.Interruption
}

func (e *Ratelimit) Init(rm rules.RuleMetadata, opts string) error {
	fmt.Println("Ratelimit plugin initiated", opts)
	var err error

	e.allowedCount, err = strconv.Atoi(opts[:len(opts)-1])
	if err != nil {
		return errors.New("invalid options for ratelimit actions")
	}

	// decides clearAfterSeconds from the last variable
	unit := opts[len(opts)-1:]
	if unit == "s" {
		e.clearAfterSeconds = 1
	} else if unit == "m" {
		e.clearAfterSeconds = 60
	} else if unit == "h" {
		e.clearAfterSeconds = 3600
	} else if unit == "d" {
		e.clearAfterSeconds = 86400
	} else {
		return errors.New("invalid options for ratelimit actions")
	}

	// Initializing the ratelimit
	e.remainingCount = e.allowedCount
	e.lastClearRatelimitTime = time.Now()

	// Generating interrupt - right now static to deny-429
	e.interrupt = &types.Interruption{
		RuleID: rm.ID(),
		Action: "deny",
		Status: 429,
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

	//check if the time is greater than the clearAfterSeconds
	if time.Since(e.lastClearRatelimitTime).Seconds() > float64(e.clearAfterSeconds) {
		fmt.Println("Ratelimit reset")
		corazaLogger.Debug().Msg("Ratelimit reset")
		e.remainingCount = e.allowedCount
		e.lastClearRatelimitTime = time.Now()
	}

	if e.remainingCount <= 0 {
		fmt.Println("Ratelimit exceeded")
		corazaLogger.Debug().Msg("Ratelimit exceeded")
		tx.Interrupt(e.interrupt)

		return
	}

	e.remainingCount--

	corazaLogger.Debug().Msg("Evaluating ratelimit plugin")
	corazaLogger.Debug().Msg(fmt.Sprintf("Hits left: %d", e.allowedCount-e.remainingCount))

	fmt.Println("Ratelimit remaining count: ", e.remainingCount)

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
