package trackooor

import (
	"context"
	"evm-trackooor/actions"
	"evm-trackooor/shared"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// FUNCTIONS

// forward data to action function if the action func mapping exists
func doEventPostProcessing(vLog types.Log) {

	contract := vLog.Address // contract that emitted the event log
	if len(vLog.Topics) == 0 {
		return
	}

	eventSigHash := vLog.Topics[0]

	var eventFields shared.EventFields
	var decodedTopics map[string]interface{}
	var decodedData map[string]interface{}
	var actionEventData actions.ActionEventData

	// if event sig is recognised, decode event log
	if event, ok := shared.EventSigs[eventSigHash.Hex()]; ok {
		// decode event log
		event := event.(map[string]interface{})

		eventFields, decodedTopics, decodedData = shared.DecodeTopicsAndData(vLog.Topics[1:], vLog.Data, event["abi"])
		actionEventData = actions.ActionEventData{
			EventLog:      vLog,
			EventFields:   eventFields,
			DecodedTopics: decodedTopics,
			DecodedData:   decodedData,
		}
	} else {
		actionEventData = actions.ActionEventData{
			EventLog: vLog,
		}
	}

	// call action funcs (if mappings exist)

	// check event sig -> func mapping
	actions.ActionMapMutex.RLock()
	if functions, ok := actions.EventSigToAction[eventSigHash]; ok {
		actions.ActionMapMutex.RUnlock()
		for _, function := range functions {
			go function(actionEventData)
		}
	} else {
		actions.ActionMapMutex.RUnlock()
	}

	// check contract address, event sig -> func mapping
	actions.ActionMapMutex.RLock()
	if functions, ok := actions.ContractToEventSigToAction[contract][eventSigHash]; ok {
		actions.ActionMapMutex.RUnlock()
		for _, function := range functions {
			go function(actionEventData)
		}
	} else {
		actions.ActionMapMutex.RUnlock()
	}

	// check contract address -> event func mapping
	actions.ActionMapMutex.RLock()
	if functions, ok := actions.ContractToEventAction[contract]; ok {
		actions.ActionMapMutex.RUnlock()

		for _, function := range functions {
			go function(actionEventData)
		}
	} else {
		actions.ActionMapMutex.RUnlock()
	}
}

func handleEventGeneral(vLog types.Log) {
	// make sure the event is not already processed when switching historical -> realtime
	if shared.SwitchingToRealtime {
		// check if already processed
		if entity, ok := shared.AlreadyProcessed[vLog.TxHash]; ok {
			if vLog.Index == entity.EventIndex {
				// event was already processed, dont process again
				shared.DupeDetected = true
				fmt.Printf("Duplicate event detected, ignoring this event\n")
				return
			}
		}
		// otherwise add to map
		shared.AlreadyProcessed[vLog.TxHash] = shared.ProcessedEntity{EventIndex: vLog.Index}
	}

	// send to post processing (if mapping exists)
	doEventPostProcessing(vLog)
}

// listens to events of multiple contracts, but instead of creating an event subscription log
// for each contract, it uses a single channel for all of them.
// downside is that event sigs cannot be filtered for each contract individually
// aka cannot filter only event X for contract A, and event Y for contract B
func ListenToEventsSingle() {

	// create single filter query with all contract addresses, and the event topics to filter by (if any)
	query := ethereum.FilterQuery{
		Addresses: shared.Options.FilterAddresses,
		Topics:    shared.Options.FilterEventTopics,
	}

	// subscribe to events
	logs := make(chan types.Log)
	// resubscribe incase webhook goes down momentarily
	sub := event.Resubscribe(2*time.Second, func(ctx context.Context) (event.Subscription, error) {
		fmt.Printf("Subscribing to event logs\n")
		return shared.Client.SubscribeFilterLogs(context.Background(), query, logs)
	})

	// terminal output that we are listening
	logListeningEventsSingle(shared.Options)

	// forever loop: handle event logs
	for {
		select {
		case err := <-sub.Err():
			log.Fatal(err)
		case vLog := <-logs:
			if vLog.Removed {
				shared.Warnf(slog.Default(), "Log reverted due to chain reorganisation, shouldn't be handling this event: %v\n", vLog.TxHash.Hex())
			} else {
				handleEventGeneral(vLog)
			}
		}
	}
}
