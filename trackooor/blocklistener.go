package trackooor

import (
	"context"
	"evm-trackooor/actions"
	"evm-trackooor/shared"
	"evm-trackooor/utils"
	"fmt"
	"log"
	"log/slog"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/schollz/progressbar/v3"
)

// handles all events emitted in a given block,
func handleEventsFromBlock(block *types.Block, contracts []common.Address) {
	// blockHash := block.Hash()
	query := ethereum.FilterQuery{
		// BlockHash: &blockHash,
		FromBlock: block.Number(),
		ToBlock:   block.Number(),
		Addresses: contracts,
		Topics:    shared.Options.FilterEventTopics,
	}
	logs, err := shared.Client.FilterLogs(context.Background(), query)
	if err != nil {
		log.Fatal(err)
	}

	var bar *progressbar.ProgressBar
	if shared.Verbose {
		bar = progressbar.Default(int64(len(logs)), "Handling events")
	}
	for _, log := range logs {
		handleEventGeneral(log)
		if shared.Verbose {
			bar.Add(1)
		}
	}
}

// call transaction action function, if exists
func doTxPostProcessing(block *types.Block) {
	var bar *progressbar.ProgressBar
	if shared.Verbose {
		bar = progressbar.Default(int64(len(block.Transactions())), "Processing transactions")
	}
	for _, tx := range block.Transactions() {
		to := tx.To()
		from := utils.GetTxSender(tx)

		// send tx data to post processing
		p := actions.ActionTxData{
			Transaction: tx,
			Block:       block,
			From:        from,
			To:          to,
		}
		// check tx `from` address, if no match then check `to` address
		actions.ActionMapMutex.RLock()
		if functions, ok := actions.TxAddressToAction[*p.From]; ok {
			// loop through and call all action funcs associated with that address
			for _, function := range functions {
				go function(p)
			}
		} else if p.To != nil { // in case tx was deployment, `To` will be null
			if functions, ok := actions.TxAddressToAction[*p.To]; ok {
				for _, function := range functions {
					go function(p)
				}
			}
		}
		actions.ActionMapMutex.RUnlock()

		if shared.Verbose {
			bar.Add(1)
		}
	}
}

func doBlocksPostProcessing(p actions.ActionBlockData) {
	for _, function := range actions.BlockActions {
		go function(p)
	}
}

func HandleHeader(header *types.Header) {
	shared.Infof(slog.Default(), "Handling header for block %v\n", header.Number)

	// make sure the event is not already processed when switching historical -> realtime
	if shared.SwitchingToRealtime {
		// check if already processed
		if entity, ok := shared.AlreadyProcessed[header.Hash()]; ok {
			if entity.BlockNumber.Cmp(header.Number) == 0 {
				// block was already processed, dont process again
				shared.DupeDetected = true
				fmt.Printf("Duplicate block detected, ignoring\n")
				return
			}
		}
		// otherwise add to map
		shared.AlreadyProcessed[header.Hash()] = shared.ProcessedEntity{BlockNumber: header.Number}
	}

	shared.Infof(slog.Default(), "Getting block by hash %v", header.Hash())

	if shared.Options.IsL2Chain {
		// try getting block by hash, otherwise by number
		blockHexNum := "0x" + header.Number.Text(16)
		block, err := shared.GetL2BlockByHexNumber(blockHexNum)
		if err != nil {
			shared.Warnf(slog.Default(), "Failed to get block by hex num: %v, ignoring block, err: %v", blockHexNum, err)
			return
		}
		handleBlock(block)
	} else {
		// try getting block by hash, otherwise by number
		block, err := shared.Client.BlockByHash(context.Background(), header.Hash())
		if err != nil {
			shared.Warnf(slog.Default(), "Failed to get block by hash %v, getting by number %v, err: %v", header.Hash(), header.Number, err)
			block, err = shared.Client.BlockByNumber(context.Background(), header.Number)
			if err != nil {
				shared.Warnf(slog.Default(), "Failed to get block by number %v, ignoring block, err: %v", header.Number, err)
				return
			}
		}

		handleBlock(block)
	}
}

func handleBlock(block *types.Block) {
	shared.Infof(slog.Default(), "Handling block %v\n", block.Number())

	// call block post processing functions
	doBlocksPostProcessing(actions.ActionBlockData{Block: block})

	// handle all events emitted in the block
	actions.ActionMapMutex.RLock()
	if len(actions.EventSigToAction) == 0 && len(actions.ContractToEventSigToAction) == 0 {
		// dont process if no event sigs mapped to action funcs
		actions.ActionMapMutex.RUnlock()
		shared.Infof(slog.Default(), "No event actions enabled - not handling")
	} else {
		actions.ActionMapMutex.RUnlock()
		handleEventsFromBlock(block, shared.Options.FilterAddresses)
	}

	// do tx post processing
	// dont process if no addresses mapped to action funcs
	actions.ActionMapMutex.RLock()
	if len(actions.TxAddressToAction) == 0 {
		actions.ActionMapMutex.RUnlock()
		shared.Infof(slog.Default(), "No tx address actions enabled - not handling")
	} else {
		actions.ActionMapMutex.RUnlock()
		doTxPostProcessing(block)
	}

	// wait for actions to finish (if the action requests it)
	shared.BlockWaitGroup.Wait()
}

func ListenToBlocks() {

	headers := make(chan *types.Header)
	// resubscribe incase webhook goes down momentarily
	sub := event.Resubscribe(2*time.Second, func(ctx context.Context) (event.Subscription, error) {
		fmt.Printf("Subscribing to new head\n")
		return shared.Client.SubscribeNewHead(context.Background(), headers)
	})

	// terminal output that we are starting
	logListeningBlocks()

	for {
		select {
		case err := <-sub.Err():
			log.Fatalf("Error: %v\n", err)
		case header := <-headers:
			HandleHeader(header)
		}
	}
}

// PENDING BLOCKS LISTENER

// whether or not a block num has been processed already
var processedBlock map[string]bool

func getPendingBlock() {
	// request pending block
	block, err := shared.GetL2BlockByHexNumber("pending")
	if err != nil {
		panic(err)
	}

	alreadyProcessed := processedBlock[block.Number().String()]
	if alreadyProcessed {
		// dont process again
		return
	}
	processedBlock[block.Number().String()] = true

	handleBlock(block)
}

func ListenToPendingBlocks() {

	processedBlock = make(map[string]bool)

	// terminal output that we are starting
	fmt.Printf("Warning: Listening to pending blocks\n")
	logListeningBlocks()

	// periodically query for pending block
	ticker := time.NewTicker(50 * time.Millisecond)
	quit := make(chan struct{})
	for {
		select {
		case <-ticker.C:
			getPendingBlock()
		case <-quit:
			ticker.Stop()
			return
		}
	}
}
