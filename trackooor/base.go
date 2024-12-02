package trackooor

import (
	"evm-trackooor/database"
	"evm-trackooor/shared"
	"fmt"
	"log/slog"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// STRUCTS

type SubscriptionLogs struct {
	subscription ethereum.Subscription
	logs         chan types.Log
}

// FUNCTIONS

func init() {
	shared.AlreadyProcessed = make(map[common.Hash]shared.ProcessedEntity)
}

func SetupListener(options shared.TrackooorOptions) {
	shared.Options = options

	fmt.Printf("Loading event signatures and blockscanners\n")
	shared.LoadFromJson("./data/event_sigs.json", &shared.EventSigs)
	shared.LoadFromJson("./data/blockscanners.json", &shared.Blockscanners)

	fmt.Printf("Loading function signatures\n")
	shared.LoadFromJson("./data/func_sigs.json", &shared.FuncSigs)

	shared.Client, shared.ChainID = shared.ConnectToRPC(options.RpcURL)

	if shared.Options.AutoFetchABI {
		addContractsEventAbi(shared.Options.FilterAddresses)
	}
}

func addContractsEventAbi(contracts []common.Address) {
	fmt.Printf("Automatically fetching ABIs from blockscanner for %v contracts (this might take a while, verbose flag -v for progress)\n", len(shared.Options.FilterAddresses))
	// go through each tracked contract and fetch its ABI, adding it to
	// the event sigs data file
	shared.Infof(slog.Default(), "Fetching ABI of tracked contracts for event signatures")
	var abiStrings []string
	for _, contactAddress := range contracts {
		shared.Infof(slog.Default(), "Automatically fetching ABI for %v", contactAddress.Hex())

		abiStr, err := database.GetAbiFromBlockscanner(
			shared.Blockscanners,
			shared.ChainID,
			contactAddress,
		)
		if err != nil {
			shared.Warnf(slog.Default(), "%v", err)
			continue
		}
		abiStrings = append(abiStrings, abiStr)
	}
	// add each abi string to data file
	for _, abiString := range abiStrings {
		database.AddEventsFromAbi(abiString, "./data/event_sigs.json")
	}
	// reload the event sigs data
	shared.EventSigs = make(map[string]interface{})
	shared.LoadFromJson("./data/event_sigs.json", &shared.EventSigs)
	fmt.Printf("Done fetching ABIs\n")
}
