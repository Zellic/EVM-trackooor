package database

import (
	"encoding/json"
	"fmt"
	"github.com/Zellic/EVM-trackooor/shared"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func AddFuncSigsFromAbiFile(abiFilepath string, outputFilepath string) {
	// open and load ABI file
	shared.Infof(slog.Default(), "Reading %v for ABI", abiFilepath)
	abiStr, _ := os.ReadFile(abiFilepath)

	// add events to events data file
	AddFuncSigsFromAbi(string(abiStr), outputFilepath)
}

func AddFuncSigsFromAbi(contractAbiString string, outputFilepath string) {
	// load tx data file
	var funcSigs map[string]interface{}
	shared.LoadFromJson("./data/func_sigs.json", &funcSigs)

	// load ABI as JSON
	var abiJson []map[string]interface{}
	err := json.Unmarshal([]byte(contractAbiString), &abiJson)
	if err != nil {
		log.Fatal(err)
	}

	shared.Infof(slog.Default(), "Extracting function signatures from ABI")
	contractAbi, _ := abi.JSON(strings.NewReader(string(contractAbiString)))
	for _, contractFunc := range contractAbi.Methods {
		funcSig := contractFunc.Sig
		funcSigHash := crypto.Keccak256([]byte(funcSig))
		funcSig4ByteHex := common.BytesToHash(funcSigHash).Hex()[:2+2*4]

		var funcAbiJson map[string]interface{}
		// get ABI of function that has name
		for _, item := range abiJson {
			name, ok := item["name"].(string)
			if !ok {
				continue
			}
			itemType := item["type"].(string)
			if name == contractFunc.Name && itemType == "function" {
				funcAbiJson = item
				break
			}
		}

		// error if didnt find (shouldnt happen)
		if len(funcAbiJson) == 0 {
			shared.Warnf(slog.Default(), "Could not find func name %v in JSON, skipping!", contractFunc.Name)
			continue
		}

		funcInfoMap := make(map[string]interface{})
		funcInfoMap["name"] = contractFunc.Name
		funcInfoMap["sig"] = contractFunc.Sig
		funcInfoMap["abi"] = funcAbiJson

		funcSigs[funcSig4ByteHex] = funcInfoMap
		fmt.Printf("Added %v - %v\n", funcSig4ByteHex, funcSig)
	}

	// write out to data file
	shared.Infof(slog.Default(), "Writing out to %v", outputFilepath)
	funcSigBytes, _ := json.Marshal(funcSigs)
	os.WriteFile(outputFilepath, funcSigBytes, 0644)
	log.Printf("Data wrote to file: %v", outputFilepath)
}
