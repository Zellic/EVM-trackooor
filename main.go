package main

import (
	"encoding/json"
	"evm-trackooor/actions"
	"evm-trackooor/database"
	"evm-trackooor/shared"
	"evm-trackooor/trackooor"
	"evm-trackooor/utils"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

// GLOBAL VARIABLES

//
// command args/flags
//

// general
var verbose bool

// trackooors
var configFilepath string // path to config file instead of specifying params on command line
var AutoFetchABI bool
var ContinueToRealtime bool
var PendingBlocks bool

// historical
var fromBlockStr string
var toBlockStr string
var stepBlocksStr string

// data
var abiFilepath string
var chainIdStr string
var dataContractAddresses string

var options shared.TrackooorOptions

var rootCmd = &cobra.Command{
	Use:    "evm-trackooor",
	Short:  "A modular tool to track anything on the EVM chain, including real-time tracking and alerts.",
	Long:   `A modular tool to track anything on the EVM chain, including real-time tracking and alerts.`,
	Args:   cobra.ExactArgs(1),
	PreRun: toggleVerbosity,
	Run:    func(cmd *cobra.Command, args []string) {},
}

var trackCmd = &cobra.Command{
	Use:    "track",
	Short:  "Track events, transactions and blocks in realtime, or fetch historically",
	Long:   ``,
	Args:   cobra.ExactArgs(1),
	PreRun: toggleVerbosity,
	Run:    func(cmd *cobra.Command, args []string) {},
}

var realtimeCmd = &cobra.Command{
	Use:    "realtime",
	Short:  "Track events, transactions and blocks in realtime",
	Long:   `Track events, transactions and blocks in realtime`,
	Args:   cobra.RangeArgs(0, 1),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		setListenerOptions()
		loadConfigFile(configFilepath)

		trackooor.SetupListener(options)
		actions.InitActions(options.Actions)

		if shared.BlockTrackingRequired {
			trackooor.ListenToBlocks()
		} else {
			trackooor.ListenToEventsSingle()
		}
	},
}

var realtimeEventsCmd = &cobra.Command{
	Use:    "events",
	Short:  "Track events only in realtime, through a filter log subscription",
	Long:   `Track events only in realtime, through a filter log subscription`,
	Args:   cobra.ExactArgs(0),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		setListenerOptions()

		loadConfigFile(configFilepath)

		trackooor.SetupListener(options)

		actions.InitActions(options.Actions)

		trackooor.ListenToEventsSingle()
	},
}

var realtimeBlocksCmd = &cobra.Command{
	Use:    "blocks",
	Short:  "Listens for newly mined blocks, to track events, transactions and blocks mined",
	Long:   `Listens for newly mined blocks, to track events, transactions and blocks mined`,
	Args:   cobra.ExactArgs(0),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		setListenerOptions()

		loadConfigFile(configFilepath)

		trackooor.SetupListener(options)

		actions.InitActions(options.Actions)

		if PendingBlocks {
			trackooor.ListenToPendingBlocks()
		} else {

			trackooor.ListenToBlocks()
		}
	},
}

var historicalCmd = &cobra.Command{
	Use:    "historical",
	Short:  "Fetches historical events/transactions/blocks in a given block range",
	Long:   `Fetches historical events/transactions/blocks in a given block range`,
	Args:   cobra.RangeArgs(0, 1),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		setListenerOptions()

		loadConfigFile(configFilepath)

		trackooor.SetupListener(options)
		trackooor.SetupHistorical()

		actions.InitActions(options.Actions)

		if shared.BlockTrackingRequired {
			trackooor.GetPastBlocks()
		} else {
			trackooor.GetPastEventsSingle()
		}
	},
}

var historicalEventsCmd = &cobra.Command{
	Use:    "events",
	Short:  "Fetches historically emitted events only in a given block range",
	Long:   `Fetches historically emitted events only in a given block range`,
	Args:   cobra.ExactArgs(0),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		setListenerOptions()

		loadConfigFile(configFilepath)

		trackooor.SetupListener(options)
		trackooor.SetupHistorical()

		actions.InitActions(options.Actions)

		trackooor.GetPastEventsSingle()
	},
}

var historicalBlocksCmd = &cobra.Command{
	Use:    "blocks",
	Short:  "Fetches historical blocks in a given block range",
	Long:   `Fetches historical blocks to process events and transactions in a given block range`,
	Args:   cobra.ExactArgs(0),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		setListenerOptions()

		loadConfigFile(configFilepath)

		trackooor.SetupListener(options)
		trackooor.SetupHistorical()

		actions.InitActions(options.Actions)

		trackooor.GetPastBlocks()
	},
}

var dataCmd = &cobra.Command{
	Use:    "data",
	Short:  "Add or modify data files such as event sigs and blockscanners",
	Long:   `Add or modify data files such as event sigs and blockscanners`,
	Args:   cobra.ExactArgs(1),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
	},
}

var addEventsCmd = &cobra.Command{
	Use:    "event",
	Short:  "Add event signatures to JSON data file from ABI file",
	Long:   `Add event signatures to JSON data file from ABI file`,
	Args:   cobra.ExactArgs(0),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		outputFilepath := "./data/event_sigs.json"

		// if abi file provided, use that for abi data
		if abiFilepath != "" {
			database.AddEventsFromAbiFile(abiFilepath, outputFilepath)
		} else {
			// otherwise contract address was provided, use that

			// get blockscanner data
			var blockscannerData map[string]interface{}
			shared.LoadFromJson("./data/blockscanners.json", &blockscannerData)

			// convert chainID to big int
			chainId := big.NewInt(0)
			chainId, ok := chainId.SetString(chainIdStr, 10)
			if !ok {
				log.Fatalf("Could not convert %v to big.Int", chainId)
			}

			// get contract addresses
			contractAddresses := strings.Split(dataContractAddresses, ",")
			// get each contract's abi string
			var abiStrings []string
			for _, hexAddr := range contractAddresses {
				if !common.IsHexAddress(hexAddr) {
					log.Fatalf("Provided address %v is not a hex address", hexAddr)
				}
				abiStr, err := database.GetAbiFromBlockscanner(
					blockscannerData,
					chainId,
					common.HexToAddress(hexAddr),
				)
				if err != nil {
					log.Fatal(err)
				}
				abiStrings = append(abiStrings, abiStr)
			}
			// add each abi string to output file
			for _, abiString := range abiStrings {
				database.AddEventsFromAbi(abiString, outputFilepath)
			}
		}
	},
}

var addFuncSigCmd = &cobra.Command{
	Use:    "func",
	Short:  "Add function signatures to JSON data file from ABI file",
	Long:   `Add function signatures to JSON data file from ABI file`,
	Args:   cobra.ExactArgs(0),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		outputFilepath := "./data/func_sigs.json"

		// if abi file provided, use that for abi data
		if abiFilepath != "" {
			database.AddFuncSigsFromAbiFile(abiFilepath, outputFilepath)
		} else {
			// otherwise contract address was provided, use that

			// get blockscanner data
			var blockscannerData map[string]interface{}
			shared.LoadFromJson("./data/blockscanners.json", &blockscannerData)

			// convert chainID to big int
			chainId := big.NewInt(0)
			chainId, ok := chainId.SetString(chainIdStr, 10)
			if !ok {
				log.Fatalf("Could not convert %v to big.Int", chainId)
			}

			// get contract addresses
			contractAddresses := strings.Split(dataContractAddresses, ",")
			// get each contract's abi string
			var abiStrings []string
			for _, hexAddr := range contractAddresses {
				if !common.IsHexAddress(hexAddr) {
					log.Fatalf("Provided address %v is not a hex address", hexAddr)
				}
				abiStr, err := database.GetAbiFromBlockscanner(
					blockscannerData,
					chainId,
					common.HexToAddress(hexAddr),
				)
				if err != nil {
					log.Fatal(err)
				}
				abiStrings = append(abiStrings, abiStr)
			}
			// add each abi string to output file
			for _, abiString := range abiStrings {
				database.AddFuncSigsFromAbi(abiString, outputFilepath)
			}
		}
	},
}

var displayActionsCmd = &cobra.Command{
	Use:    "actions",
	Short:  "Displays available actions (post processors) and their usage",
	Long:   `Displays available actions (post processors) and their usage`,
	Args:   cobra.RangeArgs(0, 1),
	PreRun: toggleVerbosity,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			actions.DisplayActions()
		} else {
			actionName := args[0]
			actions.DisplayActionInfo(actionName)
		}
	},
}

// FUNCTIONS

func toggleVerbosity(cmd *cobra.Command, args []string) {
	shared.Verbose = verbose
	if verbose {
		slog.SetLogLoggerLevel(slog.LevelInfo)
	} else {
		slog.SetLogLoggerLevel(slog.LevelWarn)
	}
}

func setListenerOptions() {
	options.AutoFetchABI = AutoFetchABI
	options.HistoricalOptions.ContinueToRealtime = ContinueToRealtime

	// set fromBlock
	if fromBlockStr != "" {
		fromBlock := big.NewInt(0)
		fromBlock, ok := fromBlock.SetString(fromBlockStr, 10)
		if !ok {
			log.Fatalf("Error when setting fromBlock, value: %v", fromBlockStr)
		}
		options.HistoricalOptions.FromBlock = fromBlock

		// set toBlock
		if toBlockStr != "" {
			toBlock := big.NewInt(0)
			toBlock, ok = toBlock.SetString(toBlockStr, 10)
			if !ok {
				log.Fatalf("Error when setting toBlock, value: %v", toBlockStr)
			}
			options.HistoricalOptions.ToBlock = toBlock

			// if fromBlock > toBlock, loop backwards
			if fromBlock.Cmp(toBlock) == 1 {
				options.HistoricalOptions.LoopBackwards = true
			} else {
				options.HistoricalOptions.LoopBackwards = false
			}
		}
	}

	// set stepBlocks
	if stepBlocksStr == "" {
		// set default value
		options.HistoricalOptions.StepBlocks = big.NewInt(10000)
	} else {
		stepBlocks := big.NewInt(0)
		stepBlocks, ok := stepBlocks.SetString(stepBlocksStr, 10)
		if !ok {
			log.Fatalf("Error when setting stepBlocks, value: %v", stepBlocksStr)
		}
		options.HistoricalOptions.StepBlocks = stepBlocks
	}
}

func loadConfigFile(filename string) {
	if filename == "" {
		log.Fatalf("No config file provided (see README.md), provide with --config <filepath>")
	}

	var configOptions map[string]interface{}
	// load config json
	shared.Infof(slog.Default(), "Reading %v for config options", filename)
	f, _ := os.ReadFile(filename)
	err := json.Unmarshal(f, &configOptions)
	if err != nil {
		log.Fatal(err)
	}
	// set options
	// RPC URL
	if v, ok := configOptions["rpcurl"]; ok {
		options.RpcURL = v.(string)
	} else {
		log.Fatalf("Missing RPC URL in config file %v", filename)
	}

	// load actions
	// init maps
	options.AddressProperties = make(map[common.Address]map[string]interface{})
	options.Actions = make(map[string]shared.ActionOptions)
	if v, ok := configOptions["actions"]; ok {
		actions := v.(map[string]interface{})
		for actionName, actionConfigInterface := range actions {
			actionConfig := actionConfigInterface.(map[string]interface{})
			// dont use action if disabled explicitly
			if v, ok := actionConfig["enabled"]; ok {
				enabled := v.(bool)
				if !enabled {
					continue
				}
			}

			var actionOptions shared.ActionOptions
			if addressesInterface, ok := actionConfig["addresses"]; ok {
				addresses := addressesInterface.(map[string]interface{})
				// set action options

				for hexAddress, addressPropertiesInterface := range addresses {
					addressProperties := addressPropertiesInterface.(map[string]interface{})
					// if address is explicitly disabled, ignore
					if enabled, ok := addressProperties["enabled"]; ok {
						if !enabled.(bool) {
							continue
						}
					}

					address := common.HexToAddress(hexAddress)
					// add address to global addresses
					options.FilterAddresses = append(options.FilterAddresses, address)
					options.AddressProperties[address] = addressProperties
					// add address to action specific addresses
					actionOptions.Addresses = append(actionOptions.Addresses, address)
				}
			}
			// action config
			if actionOptionsInterface, ok := actionConfig["options"]; ok {
				actionCustomOptions := actionOptionsInterface.(map[string]interface{})
				actionOptions.CustomOptions = actionCustomOptions
			}
			options.Actions[actionName] = actionOptions
		}
		// remove duplicate addresses
		options.FilterAddresses = utils.RemoveDuplicates(options.FilterAddresses)
	} else {
		log.Fatalf("No actions specified in config file")
	}

	// get event sigs to filter for
	// event sigs are applied for all contracts, cannot apply individually
	if v, ok := configOptions["event-signatures"]; ok {
		eventSigs := v.([]interface{})
		shared.LoadFromJson("./data/event_sigs.json", &shared.EventSigs) // (see below)
		for _, eventSigInterface := range eventSigs {
			eventSig := eventSigInterface.(string)
			eventSigHash := crypto.Keccak256Hash([]byte(eventSig))

			// if filtering by event signatures, make sure that these
			// signatures actually exist in the event signatures data file
			// otherwise they are unknown events (no ABI to decode them)
			_, ok := shared.EventSigs[eventSigHash.Hex()]
			if !ok {
				log.Fatalf("Event ABI for '%v', does not exist, please add its ABI to the event signatures data file", eventSig)
			}
			// event sig is topic[0]
			if len(options.FilterEventTopics) <= 0 {
				options.FilterEventTopics = append(options.FilterEventTopics, []common.Hash{})
			}
			options.FilterEventTopics[0] = append(options.FilterEventTopics[0], eventSigHash)
		}
	}

	// get event topics to filter for
	// event sigs are technically just topic[0], so we will aggregate these together
	if v, ok := configOptions["event-topics"]; ok {
		eventTopics := v.([]interface{})
		for ind, eventTopicsInt := range eventTopics {
			topicInterfaces := eventTopicsInt.([]interface{})
			var topicHashes []common.Hash
			for _, topicInt := range topicInterfaces {
				topic := topicInt.(string)
				topicHash := common.HexToHash(topic)
				topicHashes = append(topicHashes, topicHash)
			}
			if len(options.FilterEventTopics) <= ind {
				options.FilterEventTopics = append(options.FilterEventTopics, []common.Hash{})
			}
			options.FilterEventTopics[ind] = append(options.FilterEventTopics[ind], topicHashes...)
		}
	}

	if v, ok := configOptions["l2"]; ok {
		options.IsL2Chain = v.(bool)
	}

	if v, ok := configOptions["max-requests-per-second"]; ok {
		options.MaxRequestsPerSecond = int(v.(float64))
	}
}

func init() {

	// parse flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Verbose output")

	// realtime
	trackCmd.PersistentFlags().BoolVar(&AutoFetchABI, "fetch-abi", false, "Automatically get contract ABI for event signatures from blockscanners")
	// trackCmd.PersistentFlags().BoolVar(&SingleSubscriptionLog, "single-sub", false, "Use one subscription log for all contracts. Cannot filter event signatures individually for each contract if doing so.")

	// specify config file with options, instead of providing through command line
	trackCmd.PersistentFlags().StringVar(&configFilepath, "config", "", "Path the config file instead of specifying settings through command line")

	// data

	// add event sigs
	addEventsCmd.PersistentFlags().StringVar(&abiFilepath, "abi", "", "Path to the abi file")
	addEventsCmd.PersistentFlags().StringVar(&dataContractAddresses, "contract", "", "Contract addresses, comma separated")
	addEventsCmd.PersistentFlags().StringVar(&chainIdStr, "chain", "", "Chain ID contract is deployed on")
	addEventsCmd.MarkFlagsOneRequired("abi", "contract")
	addEventsCmd.MarkFlagsMutuallyExclusive("abi", "contract")
	addEventsCmd.MarkFlagsRequiredTogether("contract", "chain")

	// add func sigs
	addFuncSigCmd.PersistentFlags().StringVar(&abiFilepath, "abi", "", "Path to the abi file")
	addFuncSigCmd.PersistentFlags().StringVar(&dataContractAddresses, "contract", "", "Contract addresses, comma separated")
	addFuncSigCmd.PersistentFlags().StringVar(&chainIdStr, "chain", "", "Chain ID contract is deployed on")
	addFuncSigCmd.MarkFlagsOneRequired("abi", "contract")
	addFuncSigCmd.MarkFlagsMutuallyExclusive("abi", "contract")
	addFuncSigCmd.MarkFlagsRequiredTogether("contract", "chain")

	// historical
	// historicalCmd.PersistentFlags().BoolVar(&ContinueToRealtime, "continue-realtime", false, "Starts listening for and processing realtime events/block after historical is done. End block range must not be set.")
	historicalCmd.PersistentFlags().StringVar(&fromBlockStr, "from-block", "", "Block to start from when doing historical tracking")
	historicalCmd.PersistentFlags().StringVar(&toBlockStr, "to-block", "", "Block to stop at (inclusive) when doing historical tracking")
	historicalCmd.PersistentFlags().StringVar(&stepBlocksStr, "step-blocks", "", "How many blocks to request at a time, only relevant for historical event filter log.")

	// listening to pending (unconfirmed) blocks
	realtimeBlocksCmd.PersistentFlags().BoolVar(&PendingBlocks, "pending-blocks", false, "Whether or not to listen for pending (unconfirmed) blocks")

	// root cmd structure
	// root
	// - track
	// 		- realtime
	//			- events
	// 			- blocks
	// 		- historical
	//			- events
	// 			- blocks
	// - data
	// 		- add
	// - actions
	rootCmd.AddCommand(trackCmd)
	rootCmd.AddCommand(dataCmd)
	rootCmd.AddCommand(displayActionsCmd)

	trackCmd.AddCommand(realtimeCmd)
	trackCmd.AddCommand(historicalCmd)
	realtimeCmd.AddCommand(realtimeEventsCmd)
	realtimeCmd.AddCommand(realtimeBlocksCmd)
	historicalCmd.AddCommand(historicalEventsCmd)
	historicalCmd.AddCommand(historicalBlocksCmd)

	dataCmd.AddCommand(addEventsCmd)
	dataCmd.AddCommand(addFuncSigCmd)
}

func executeArgs() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func main() {
	executeArgs()
}
