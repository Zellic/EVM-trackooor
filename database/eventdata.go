package database

import (
	"encoding/json"
	"evm-trackooor/shared"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// adds event data and abi to event data file
func AddEventsFromAbiFile(abiFilepath string, outputFilepath string) {

	// open and load ABI file
	shared.Infof(slog.Default(), "Reading %v for ABI", abiFilepath)
	abiStr, _ := os.ReadFile(abiFilepath)

	// add events to events data file
	AddEventsFromAbi(string(abiStr), outputFilepath)
}

func AddEventsFromAbi(contractAbiString string, outputFilepath string) {
	// the reason we take a string is that marshalling abi.ABI causes
	// keys to get uppercased (due to how go handles internal/external variables)
	// so by providing a string, we retain the JSON to extract event abi
	// from there afterwards
	// TODO(actually i think this can be fixed with `json` struct tags)

	// load events data file
	var eventSigs map[string]interface{}
	shared.LoadFromJson("./data/event_sigs.json", &eventSigs)

	// load ABI as JSON
	var abiJson []map[string]interface{}
	err := json.Unmarshal([]byte(contractAbiString), &abiJson)
	if err != nil {
		log.Fatal(err)
	}

	// load ABI as abi.ABI
	shared.Infof(slog.Default(), "Extracting events from ABI")
	contractAbi, _ := abi.JSON(strings.NewReader(string(contractAbiString)))

	for _, event := range contractAbi.Events {
		eventInfoMap := make(map[string]interface{})
		eventSigHash := crypto.Keccak256([]byte(event.Sig))
		eventSigHashHex := common.BytesToHash(eventSigHash)

		// get event ABI from JSON ABI (reasons)
		var eventAbiJson map[string]interface{}
		for _, item := range abiJson {
			name, ok := item["name"].(string)
			if !ok {
				continue
			}
			itemType := item["type"].(string)
			if name == event.Name && itemType == "event" {
				eventAbiJson = item
				break
			}
		}
		// error if didnt find (shouldnt happen)
		if len(eventAbiJson) == 0 {
			shared.Warnf(slog.Default(), "Could not find event name %v in JSON, skipping!", event.Name)
			continue
		}

		eventInfoMap["name"] = event.Name
		eventInfoMap["sig"] = event.Sig
		eventInfoMap["abi"] = eventAbiJson

		eventSigs[eventSigHashHex.Hex()] = eventInfoMap
		fmt.Printf("Added %v\n", event.Sig)
	}

	// write out to data file
	shared.Infof(slog.Default(), "Writing out to %v", outputFilepath)
	eventSigsBytes, _ := json.Marshal(eventSigs)
	os.WriteFile(outputFilepath, eventSigsBytes, 0644)
	log.Printf("Data wrote to file: %v", outputFilepath)
}

func GetAbiFromBlockscanner(blockscannerData map[string]interface{}, chainID *big.Int, contractAddress common.Address) (string, error) {
	shared.Infof(slog.Default(), "Getting contract %v ABI from blockscanner", contractAddress.Hex())

	chain, ok := blockscannerData[chainID.String()]
	if !ok {
		return "", fmt.Errorf("could not find blockscanner data for chain %v", chainID.String())
	}
	chainMap := chain.(map[string]interface{})
	url, ok := chainMap["contract-abi"]
	if !ok {
		return "", fmt.Errorf("could not find blockscanner contract abi data for chain %v", chainID.String())
	}
	urlStr := url.(string)
	urlStr = fmt.Sprintf(urlStr, contractAddress.Hex())

	// fetch abi with http get
	resp, err := http.Get(urlStr)
	if err != nil {
		log.Fatalln(err)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalln(err)
	}
	abiStr := string(body)
	if strings.Contains(abiStr, "Contract source code not verified") {
		return "", fmt.Errorf("contract source code not verified for %v", contractAddress.Hex())
	}
	if strings.Contains(abiStr, "Max rate limit reached, please use API Key for higher rate limit") {
		// rate limited, trying again in 5 seconds
		shared.Infof(slog.Default(), "Rate limited, trying again in 5 seconds")
		time.Sleep(5 * time.Second)
		return GetAbiFromBlockscanner(blockscannerData, chainID, contractAddress)
	}
	if strings.Contains(abiStr, "Missing/Invalid API Key") {
		return "", fmt.Errorf("invalid API key: %v", abiStr)
	}
	return abiStr, nil
}
