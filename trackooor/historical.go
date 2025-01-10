package trackooor

import (
	"context"
	"evm-trackooor/actions"
	"evm-trackooor/shared"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/schollz/progressbar/v3"
)

// default values
var stepBlocks = big.NewInt(10000)
var MaxRequestsPerSecond = 30 // max rpc requests per second

// function called just before starting a historical processor
func SetupHistorical() {
	if shared.Options.HistoricalOptions.StepBlocks != nil {
		stepBlocks = shared.Options.HistoricalOptions.StepBlocks
	}
	if shared.Options.MaxRequestsPerSecond != 0 {
		MaxRequestsPerSecond = shared.Options.MaxRequestsPerSecond
	}

	if shared.Options.HistoricalOptions.FromBlock == nil {
		log.Fatalf("from-block not set!")
	}
	// use latest block if toBlock not set
	if shared.Options.HistoricalOptions.ToBlock == nil {
		header, err := shared.Client.HeaderByNumber(context.Background(), nil)
		if err != nil {
			panic(err)
		}
		shared.Options.HistoricalOptions.ToBlock = header.Number
	}
}

func ProcessHistoricalEventsSingle(fromBlock *big.Int, toBlock *big.Int, eventTopics [][]common.Hash) {
	// progress bar
	var mainBar *progressbar.ProgressBar
	mainBar = progressbar.Default(big.NewInt(0).Div(big.NewInt(0).Sub(toBlock, fromBlock), stepBlocks).Int64())

	// loop through whole block range, requesting a certain amount of blocks at once
	for currentBlock := fromBlock; currentBlock.Cmp(toBlock) <= 0; currentBlock.Add(currentBlock, stepBlocks) {
		stepFrom := currentBlock
		// currentBlock + stepBlocks - 1 , due to block ranges being inclusive on both ends
		stepTo := big.NewInt(0).Add(currentBlock, big.NewInt(0).Sub(stepBlocks, big.NewInt(1)))
		// don't step past `toBlock`
		if stepTo.Cmp(toBlock) == 1 {
			stepTo = toBlock
		}

		// create single filter query with all contracts, using same event sig filter for all
		query := ethereum.FilterQuery{
			Addresses: shared.Options.FilterAddresses,
			Topics:    eventTopics,
			FromBlock: stepFrom,
			ToBlock:   stepTo,
		}

		// request event logs filtered by filter log
		logs, err := shared.Client.FilterLogs(context.Background(), query)
		if err != nil {
			log.Fatal(err)
		}

		// handle returned logs
		var bar *progressbar.ProgressBar
		if shared.Verbose {
			bar = progressbar.Default(int64(len(logs)), fmt.Sprintf("%v events from blocks %v-%v", len(logs), stepFrom, stepTo))
		}
		for _, log := range logs {
			handleEventGeneral(log)

			// handle request throughput
			if actions.RpcWaitGroup.GetCount() >= MaxRequestsPerSecond {
				time.Sleep(time.Second)
				actions.RpcWaitGroup.Wait()
			}
			if shared.Verbose {
				bar.Add(1)
			}
		}
		mainBar.Add(1)
	}
}

func ProcessHistoricalBlocks(fromBlock *big.Int, toBlock *big.Int) {
	// progress bar
	var mainBar *progressbar.ProgressBar
	mainBar = progressbar.Default(big.NewInt(0).Sub(toBlock, fromBlock).Int64())

	if shared.Options.HistoricalOptions.BatchFetchBlocks {
		// process one block at a time
		if shared.Options.HistoricalOptions.LoopBackwards {
			log.Fatalf("unimplemented LoopBackwards & BatchFetchBlocks")
		} else {
			batchCount := big.NewInt(100)
			zero := big.NewInt(0)

			var blockNumWaitGroup sync.WaitGroup
			for currentBlock := fromBlock; currentBlock.Cmp(toBlock) <= 0; currentBlock.Add(currentBlock, big.NewInt(1)) {
				mainBar.Describe(fmt.Sprintf("Block %v", currentBlock))

				blockNumWaitGroup.Add(1)
				shared.BlockWaitGroup.Add(1)
				go func() {
					tmpCurBlock := big.NewInt(0).Set(currentBlock)
					blockNumWaitGroup.Done()

					block, err := shared.Client.BlockByNumber(context.Background(), tmpCurBlock)
					if err != nil {
						log.Fatalf("Could not retrieve block %v, err: %v\n", tmpCurBlock, err)
					}

					handleBlock(block)
				}()

				blockNumWaitGroup.Wait()

				if big.NewInt(0).Mod(currentBlock, batchCount).Cmp(zero) == 0 {
					// fmt.Printf("currentBlock: %v, waiting for BlockWaitGroup...\n", currentBlock) // DEBUG
					shared.BlockWaitGroup.Wait()
				}

				mainBar.Add(1)
			}
		}
	} else {
		// process one block at a time
		if shared.Options.HistoricalOptions.LoopBackwards {
			shared.Infof(slog.Default(), "Looping backwards from %v to %v\n", fromBlock, toBlock)
			mainBar = progressbar.Default(big.NewInt(0).Sub(fromBlock, toBlock).Int64())
			for currentBlock := fromBlock; currentBlock.Cmp(toBlock) >= 0; currentBlock.Sub(currentBlock, big.NewInt(1)) {
				mainBar.Describe(fmt.Sprintf("Block %v", currentBlock))

				block, err := shared.Client.BlockByNumber(context.Background(), currentBlock)
				if err != nil {
					log.Fatalf("Could not retrieve block %v, err: %v\n", currentBlock, err)
				}
				// fmt.Printf("Retrieved block %v\n", block.Number())

				handleBlock(block)

				mainBar.Add(1)
			}
		} else {
			for currentBlock := fromBlock; currentBlock.Cmp(toBlock) <= 0; currentBlock.Add(currentBlock, big.NewInt(1)) {
				mainBar.Describe(fmt.Sprintf("Block %v", currentBlock))

				block, err := shared.Client.BlockByNumber(context.Background(), currentBlock)
				if err != nil {
					log.Fatalf("Could not retrieve block %v, err: %v\n", currentBlock, err)
				}

				handleBlock(block)

				mainBar.Add(1)
			}
		}
	}
}

func GetPastEventsSingle() {

	// output starting message to terminal
	logHistoricalEventsSingle()

	// block range (make new variable to prevent actual values from being modified as they are passed as pointers)
	fromBlock := big.NewInt(0).Set(shared.Options.HistoricalOptions.FromBlock)
	toBlock := big.NewInt(0).Set(shared.Options.HistoricalOptions.ToBlock)

	ProcessHistoricalEventsSingle(fromBlock, toBlock, shared.Options.FilterEventTopics)

	// TODO figure if want to keep continue to realtime or leave that up to actions
	// // if continuing to process realtime
	// if options.HistoricalOptions.ContinueToRealtime {
	// 	// note: post processor finished functions will not be called
	// 	switchToRealtimeEventsSingle(eventSigHashes)
	// }

	shared.BlockWaitGroup.Wait()
	shared.Infof(slog.Default(), "Running post processing finished functions")
	actions.FinishActions(shared.Options.Actions)
}

func GetPastBlocks() {

	// output starting msg to terminal
	logHistoricalBlocks()

	// block range (make new variable to prevent actual values from being modified as they are passed as pointers)
	fromBlock := big.NewInt(0).Set(shared.Options.HistoricalOptions.FromBlock)
	toBlock := big.NewInt(0).Set(shared.Options.HistoricalOptions.ToBlock)

	ProcessHistoricalBlocks(fromBlock, toBlock)

	// TODO figure if want to keep continue to realtime or leave that up to actions
	// // if continuing to process realtime
	// if shared.Options.HistoricalOptions.ContinueToRealtime {
	// 	switchToRealtimeBlocks()
	// }

	shared.BlockWaitGroup.Wait()
	shared.Infof(slog.Default(), "Running post processing finished functions")
	actions.FinishActions(shared.Options.Actions)
}
