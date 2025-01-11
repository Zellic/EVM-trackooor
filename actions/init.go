package actions

import (
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"reflect"
	"slices"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/Zellic/EVM-trackooor/shared"
	"github.com/Zellic/EVM-trackooor/utils"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type ActionEventData struct {
	EventLog      types.Log
	EventFields   shared.EventFields
	DecodedTopics map[string]interface{}
	DecodedData   map[string]interface{}
}

type ActionTxData struct {
	Transaction *types.Transaction
	From        *common.Address
	To          *common.Address
	Block       *types.Block // block which the tx was in
}

type ActionBlockData struct {
	Block *types.Block
}

// map event sig hash to action functions
var EventSigToAction map[common.Hash][]func(ActionEventData)

// map contract address and event sig to functions
var ContractToEventSigToAction map[common.Address]map[common.Hash][]func(ActionEventData)

// map contract address to event handling action function
var ContractToEventAction map[common.Address][]func(ActionEventData)

// map transaction `to` or `from` address to action functions
var TxAddressToAction map[common.Address][]func(ActionTxData)

// array of action functions, all of which are called when block is mined
var BlockActions []func(ActionBlockData)

type WaitGroupCount struct {
	sync.WaitGroup
	count int64
}

func (wg *WaitGroupCount) Add(delta int) {
	atomic.AddInt64(&wg.count, int64(delta))
	wg.WaitGroup.Add(delta)
}

func (wg *WaitGroupCount) Done() {
	atomic.AddInt64(&wg.count, -1)
	wg.WaitGroup.Done()
}

func (wg *WaitGroupCount) GetCount() int {
	return int(atomic.LoadInt64(&wg.count))
}

// wait group used to track and wait for rpc requests
// if RpcWaitGroup.GetCount() is too many, historical events listener calls RpcWaitGroup.Wait()
var RpcWaitGroup WaitGroupCount

func init() {
	// init maps
	EventSigToAction = make(map[common.Hash][]func(ActionEventData))
	ContractToEventSigToAction = make(map[common.Address]map[common.Hash][]func(ActionEventData))
	ContractToEventAction = make(map[common.Address][]func(ActionEventData))
	TxAddressToAction = make(map[common.Address][]func(ActionTxData))

	shared.CachedBlocks = make(map[*big.Int]*types.Header)
}

// initializes all actions
// by calling all method functions of action struct
// that are enabled in config file
func InitActions(actions map[string]shared.ActionOptions) {
	// list of all methods of actionInit
	var actionNames []string

	Struct := action{}
	StructType := reflect.TypeOf(Struct)
	// loop through all methods of actionInit
	for i := 0; i < StructType.NumMethod(); i++ {
		method := StructType.Method(i)
		actionName := utils.TrimFirstInstance(method.Name, "Init")

		if strings.HasPrefix(method.Name, "Init") {
			// add to action names if is a init function
			actionNames = append(actionNames, actionName)

			if actionOptions, ok := actions[actionName]; ok && // its name exists in the Actions -> ActionOptions map
				strings.HasPrefix(method.Name, "Init") && // func name begins with Init
				method.PkgPath == "" { // check if method is exported (uppercase)
				// call action init func with options
				fmt.Printf("Initializing action: %v\n", method.Name)
				// shared.Infof(slog.Default(), "PostProcessing init: calling %v()\n", method.Name)
				method.Func.Call([]reflect.Value{reflect.ValueOf(action{
					o: actionOptions,
				},
				)})
			}
		}

	}

	// make sure all actions in config actually exist
	// by looping through all actions in config, and making sure a method exists
	for actionName := range actions {
		if !slices.Contains(actionNames, actionName) {
			log.Fatalf("Action '%v' does not exist, but was specified in config!", actionName)
		}
	}

	// determine whether or not we track blocks or events
	if len(TxAddressToAction) == 0 && len(BlockActions) == 0 {
		fmt.Printf("Auto determined to only track events\n")
		shared.BlockTrackingRequired = false
	} else {
		fmt.Printf("Auto determined to track blocks\n")
		shared.BlockTrackingRequired = true
	}
}

// call all action finish functions
// only called by historical tracking after historical finishes
func FinishActions(actions map[string]shared.ActionOptions) {
	Struct := action{}
	StructType := reflect.TypeOf(Struct)
	// loop through all methods of actionInit
	for i := 0; i < StructType.NumMethod(); i++ {
		method := StructType.Method(i)
		// check if enabled in config
		actionName := utils.TrimFirstInstance(method.Name, "Finished")
		if actionOptions, ok := actions[actionName]; ok && // its name exists in the actionOptions map
			strings.HasPrefix(method.Name, "Finished") && // func name begins with Init
			method.PkgPath == "" { // check if method is exported (uppercase)
			// call action finished func with options
			shared.Infof(slog.Default(), "Action finished: calling %v()\n", method.Name)
			method.Func.Call([]reflect.Value{reflect.ValueOf(action{
				o: actionOptions,
			},
			)})
		}
	}
}

// display info about all actions

const (
	Reset     = "\033[0m"
	Bold      = "\033[1m"
	Underline = "\033[4m"
	Italic    = "\033[3m"
)

func DisplayActions() {
	Struct := actionInfo{}
	StructType := reflect.TypeOf(Struct)
	// loop through all methods of actionInfo
	for i := 0; i < StructType.NumMethod(); i++ {
		method := StructType.Method(i)
		if strings.HasPrefix(method.Name, "Info") && // func name begins with Info
			method.PkgPath == "" { // check if method is exported (uppercase)
			// retrieve action info
			reflectedValues := method.Func.Call([]reflect.Value{reflect.ValueOf(actionInfo{})})
			reflectedStruct := reflectedValues[0]
			// unpack into struct
			var info actionInfo
			structValue := reflect.ValueOf(&info).Elem()
			for i := 0; i < reflectedStruct.NumField(); i++ {
				fieldName := reflectedStruct.Type().Field(i).Name
				fieldValue := reflectedStruct.Field(i)
				actionField := structValue.FieldByName(fieldName)

				if actionField.IsValid() && actionField.CanSet() {
					actionField.Set(fieldValue)
				}
			}
			// display info
			fmt.Printf(
				Underline+Bold+"\n%v\n"+Reset+
					"%v\n",
				info.ActionName,
				info.ActionOverview,
			)
		}
	}
	fmt.Printf(Italic + "\nRun actions <search_term> for more info about specific actions." + Reset)
}

// display full action info (including options) of actions that contain the given action name substring
// case insensitive
func DisplayActionInfo(actionNameSubstr string) {
	Struct := actionInfo{}
	StructType := reflect.TypeOf(Struct)

	found := false
	// loop through all methods of actionInfo
	for i := 0; i < StructType.NumMethod(); i++ {
		method := StructType.Method(i)
		if strings.HasPrefix(method.Name, "Info") && // func name begins with Info
			method.PkgPath == "" { // check if method is exported (uppercase)
			// retrieve action info
			reflectedValues := method.Func.Call([]reflect.Value{reflect.ValueOf(actionInfo{})})
			reflectedStruct := reflectedValues[0]
			// unpack into struct
			var info actionInfo
			structValue := reflect.ValueOf(&info).Elem()
			for i := 0; i < reflectedStruct.NumField(); i++ {
				fieldName := reflectedStruct.Type().Field(i).Name
				fieldValue := reflectedStruct.Field(i)
				actionField := structValue.FieldByName(fieldName)

				if actionField.IsValid() && actionField.CanSet() {
					actionField.Set(fieldValue)
				}
			}
			// display info if name matches
			if strings.Contains(strings.ToLower(info.ActionName), strings.ToLower(actionNameSubstr)) {
				fmt.Printf(
					Underline+Bold+"\n%v\n"+Reset+
						"%v\n"+
						"%v\n"+
						Bold+"\nOptions:\n"+Reset+
						"%v\n"+
						Bold+"\nExample config:\n"+Reset+
						"%v\n",
					info.ActionName,
					info.ActionOverview,
					info.ActionDescription,
					info.ActionOptionDetails,
					info.ActionConfigExample,
				)
				found = true
			}
		}
	}
	if !found {
		fmt.Printf("No actions with descriptions found for '%v', maybe this action doesn't have a description written?\n", actionNameSubstr)
	}
}
