package shared

import (
	"encoding/json"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type EventField struct {
	FieldName string
	FieldType string
	Indexed   bool
}

type EventFields []EventField

func DecodeTopicsAndData(topics []common.Hash, data []byte, _abi interface{}) (EventFields, map[string]interface{}, map[string]interface{}) {
	// get event abi
	abiMap := _abi.(map[string]interface{})
	eventName := abiMap["name"].(string)
	jsonBytes, _ := json.Marshal(_abi)
	jsonStr := string(jsonBytes)
	eventAbi, _ := abi.JSON(strings.NewReader("[" + string(jsonStr) + "]"))
	eventInfo := eventAbi.Events[eventName]

	// get event field names & types (in order)
	var eventFields EventFields
	inputs := abiMap["inputs"].([]interface{})
	for _, _input := range inputs {
		input := _input.(map[string]interface{})
		eventFields = append(eventFields, EventField{
			FieldName: input["name"].(string),
			FieldType: input["type"].(string),
			Indexed:   input["indexed"].(bool),
		})
	}

	// decode topics
	indexed := make([]abi.Argument, 0)
	for _, input := range eventInfo.Inputs {
		if input.Indexed {
			indexed = append(indexed, input)
		}
	}
	decodedTopics := make(map[string]interface{})
	abi.ParseTopicsIntoMap(decodedTopics, indexed, topics)

	// decode data
	decodedData := make(map[string]interface{})
	eventInfo.Inputs.UnpackIntoMap(decodedData, data)

	// fmt.Printf("decodedTopics: %v\n", decodedTopics)
	// fmt.Printf("decodedData: %v\n", decodedData)
	return eventFields, decodedTopics, decodedData
}

// returns function signature, and map function parameter name to value (argument)
func DecodeTransaction(funcSelectorHex string, txData []byte, _abi interface{}) (string, map[string]interface{}, error) {
	abiMap := _abi.(map[string]interface{})
	funcSig := abiMap["sig"].(string)
	jsonBytes, _ := json.Marshal(abiMap["abi"])
	jsonStr := string(jsonBytes)
	funcAbi, _ := abi.JSON(strings.NewReader("[" + string(jsonStr) + "]"))

	method, err := funcAbi.MethodById(common.Hex2Bytes(funcSelectorHex[2:]))
	if err != nil {
		return funcSig, nil, err
	}

	var args = make(map[string]interface{})
	method.Inputs.UnpackIntoMap(args, txData[4:])

	return funcSig, args, nil
}
