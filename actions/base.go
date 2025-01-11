package actions

import (
	"log"
	"log/slog"
	"reflect"
	"slices"
	"sync"

	"github.com/Zellic/EVM-trackooor/shared"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

type action struct {
	o shared.ActionOptions
}

type actionInfo struct {
	ActionName          string
	ActionOverview      string
	ActionDescription   string
	ActionOptionDetails string
	ActionConfigExample string
}

// prevent concurrent map writes for maps:
// EventSigToAction
// ContractToEventSigToAction
// TxAddressToAction
var ActionMapMutex sync.RWMutex

// add an action function to the event sig -> func mapping
func addEventSigAction(sig string, f func(ActionEventData)) {
	// ensure ABI for event sig exists
	sighash := crypto.Keccak256Hash([]byte(sig))
	_, ok := shared.EventSigs[sighash.Hex()]
	if !ok {
		log.Fatalf("Event ABI for '%v', does not exist, please add its ABI to the event signatures data file", sig)
	}

	ActionMapMutex.Lock()

	// make sure action func is not already in array
	if !containsActionEventFunc(EventSigToAction[sighash], f) {
		EventSigToAction[sighash] = append(EventSigToAction[sighash], f)
	}

	// add event sig to filter topics if not already added
	if len(shared.Options.FilterEventTopics) == 0 {
		shared.Options.FilterEventTopics = [][]common.Hash{{}}
	}
	if !slices.Contains(shared.Options.FilterEventTopics[0], sighash) {
		shared.Options.FilterEventTopics[0] = append(shared.Options.FilterEventTopics[0], sighash)
	}

	ActionMapMutex.Unlock()
}

// add an action function to the contract address, event sig -> func mapping
func addAddressEventSigAction(contractAddress common.Address, sig string, f func(ActionEventData)) {
	// ensure ABI for event sig exists
	sighash := crypto.Keccak256Hash([]byte(sig))
	_, ok := shared.EventSigs[sighash.Hex()]
	if !ok {
		log.Fatalf("Event ABI for '%v', does not exist, please add its ABI to the event signatures data file", sig)
	}

	// check if inner map is already initialized. if not, initialise it
	ActionMapMutex.Lock()
	if ContractToEventSigToAction[contractAddress] == nil {
		ContractToEventSigToAction[contractAddress] = make(map[common.Hash][]func(ActionEventData))
	}

	// make sure action func is not already in array
	if !containsActionEventFunc(ContractToEventSigToAction[contractAddress][sighash], f) {
		ContractToEventSigToAction[contractAddress][sighash] = append(ContractToEventSigToAction[contractAddress][sighash], f)
	} else {
		shared.Infof(slog.Default(), "addAddressEventSigAction - ignoring since already in array")
	}

	// add contract to filter addresses if not already added
	if !slices.Contains(shared.Options.FilterAddresses, contractAddress) {
		shared.Options.FilterAddresses = append(shared.Options.FilterAddresses, contractAddress)
	}

	// add event sig to filter topics if not already added
	if len(shared.Options.FilterEventTopics) == 0 {
		shared.Options.FilterEventTopics = [][]common.Hash{{}}
	}
	if !slices.Contains(shared.Options.FilterEventTopics[0], sighash) {
		shared.Options.FilterEventTopics[0] = append(shared.Options.FilterEventTopics[0], sighash)
	}

	ActionMapMutex.Unlock()
}

// not to be confused with addAddressEventSigAction
// this function adds mapping for contract -> event function
// it sends all events to the function, regardless of event sig
func addAddressEventAction(contractAddress common.Address, f func(ActionEventData)) {
	ActionMapMutex.Lock()
	// make sure action func is not already in array
	if !containsActionEventFunc(ContractToEventAction[contractAddress], f) {
		// add to map
		ContractToEventAction[contractAddress] = append(ContractToEventAction[contractAddress], f)
	} else {
		shared.Infof(slog.Default(), "addAddressEventAction - ignoring since already in array")
	}

	ActionMapMutex.Unlock()
}

// add an action function for transactions, from/to address -> function
func addTxAddressAction(address common.Address, f func(ActionTxData)) {
	ActionMapMutex.Lock()
	// make sure action func is not already in array
	if !containsActionTxFunc(TxAddressToAction[address], f) {
		// add to map
		TxAddressToAction[address] = append(TxAddressToAction[address], f)
	} else {
		shared.Infof(slog.Default(), "addTxAddressAction - ignoring since already in array")
	}
	ActionMapMutex.Unlock()
}

// add an action function for blocks
func addBlockAction(f func(ActionBlockData)) {
	// make sure action func is not already in array
	if !containsActionBlockFunc(BlockActions, f) {
		BlockActions = append(BlockActions, f)
	} else {
		shared.Infof(slog.Default(), "addBlockAction - ignoring since already in array")
	}
}

// helpers

// helpers to check whether or not array of funcs contains a func
// (yes, 3 separate funcs for each data type. bad, but otherwise more reflection is required which is worse)

func containsActionEventFunc(funcs []func(ActionEventData), target func(ActionEventData)) bool {
	for _, f := range funcs {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(target).Pointer() {
			return true
		}
	}
	return false
}

func containsActionTxFunc(funcs []func(ActionTxData), target func(ActionTxData)) bool {
	for _, f := range funcs {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(target).Pointer() {
			return true
		}
	}
	return false
}

func containsActionBlockFunc(funcs []func(ActionBlockData), target func(ActionBlockData)) bool {
	for _, f := range funcs {
		if reflect.ValueOf(f).Pointer() == reflect.ValueOf(target).Pointer() {
			return true
		}
	}
	return false
}
