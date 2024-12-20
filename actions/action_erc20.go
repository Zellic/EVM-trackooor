package actions

import (
	discordwebhook "evm-trackooor/discord-webhook"
	"evm-trackooor/shared"
	"evm-trackooor/utils"
	"fmt"
	"log/slog"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// tracking volume of ERC20 token, for last X minutes
const erc20VolumeTime = 5 * 60

// how often the webhook logs erc20 token volume
const logERC20VolumeFreq = 1 * 60

var erc20Volume map[common.Address]*big.Int
var erc20Transfers map[common.Address][]ERC20TransferInfo
var erc20Mutex = sync.RWMutex{} // prevent race condition for maps erc20Volume and erc20Transfers

type ERC20TransferInfo struct {
	from      common.Address
	to        common.Address
	value     *big.Int
	decimals  uint8
	symbol    string
	timestamp uint64
}

// init function called just before starting trackooor
func (action) InitERC20Events() {
	erc20Volume = make(map[common.Address]*big.Int)
	erc20Transfers = make(map[common.Address][]ERC20TransferInfo)

	// log the volume of erc20 tokens every X minutes
	go func() {
		periodicLogERC20Volumes(shared.Options.FilterAddresses)
	}()

	// retrieve info for each ERC20 token
	for _, contractAddress := range shared.Options.FilterAddresses {
		shared.ERC20TokenInfos[contractAddress] = shared.RetrieveERC20Info(contractAddress)
	}

	addEventSigAction(
		"Transfer(address,address,uint256)",
		ProcessERC20Transfer,
	)
}

func SendWebhookMsgERC20Transfer(eventLog types.Log, erc20 ERC20TransferInfo) {
	var discordEmbedMsg discordwebhook.Embed
	discordEmbedMsg.Title = ":money_with_wings: ERC20 Transfer :money_with_wings:"
	discordEmbedAuthor := discordwebhook.Author{
		Name: eventLog.Address.Hex(),
	}
	fromUrlStr := utils.BlockscanFormat("address", eventLog.Address.Hex())
	// if we managed to format the URL (otherwise embed will fail to send)
	if strings.HasPrefix(fromUrlStr, "http") {
		discordEmbedAuthor.Url = fromUrlStr
	}
	discordEmbedMsg.Author = discordEmbedAuthor
	TxUrlStr := utils.BlockscanFormat("transaction", eventLog.TxHash.Hex())
	if strings.HasPrefix(TxUrlStr, "http") {
		discordEmbedMsg.Url = TxUrlStr
	}
	discordEmbedMsg.Description = fmt.Sprintf(
		"from: %v\n"+
			"to: %v\n"+
			"value: %v (%v) %v (%v decimals)\n",
		utils.FormatBlockscanHyperlink("address", erc20.from.Hex(), erc20.from.Hex()),
		utils.FormatBlockscanHyperlink("address", erc20.to.Hex(), erc20.to.Hex()),
		utils.CodeQuote(erc20.value.String()),
		utils.CodeQuote(utils.FormatDecimals(erc20.value, erc20.decimals)),
		erc20.symbol,
		erc20.decimals,
	)
	discordEmbedMsg.Timestamp = time.Now()

	// unique colour based on event signature hash
	eventSigHash := eventLog.Topics[0]
	colourHex := eventSigHash.Hex()[2 : 2+6]
	colourNum, _ := strconv.ParseUint(colourHex, 16, 64)
	discordEmbedMsg.Color = int(colourNum)

	err := shared.DiscordWebhook.SendEmbedMessages([]discordwebhook.Embed{discordEmbedMsg})
	if err != nil {
		shared.Warnf(slog.Default(), "Webhook error: %v", err)
	}
}

func SendWebhookMsgERC20Volume(erc20Token common.Address, minutes int) {
	tokenInfo := shared.RetrieveERC20Info(erc20Token)

	erc20Mutex.RLock()
	amount := erc20Volume[erc20Token]
	erc20Mutex.RUnlock()

	shared.DiscordWebhook.SendMessage(
		fmt.Sprintf(
			"%v Volume in last %v mins\n"+
				"%v (%v) %v\n",
			tokenInfo.Name,
			minutes/60,
			utils.CodeQuote(amount.String()),
			utils.CodeQuote(utils.FormatDecimals(amount, tokenInfo.Decimals)),
			tokenInfo.Symbol,
		),
	)
}

func periodicLogERC20Volumes(tokens []common.Address) {
	ticker := time.NewTicker(logERC20VolumeFreq * time.Second)
	for range ticker.C {
		shared.DiscordWebhook.SendMessage(":dollar: ERC20 Volumes :dollar:")
		for _, token := range tokens {
			SendWebhookMsgERC20Volume(token, erc20VolumeTime)
		}
	}
}

func updateERC20Volume(token common.Address) {
	// calculate erc20 volume in last X minutes
	volume := big.NewInt(0)
	currentTimestamp := time.Now().Unix()
	var removing []int

	erc20Mutex.RLock()
	for ind, transfer := range erc20Transfers[token] {
		// mark for removal if not during last X mins
		if uint64(currentTimestamp)-transfer.timestamp > erc20VolumeTime {
			removing = append(removing, ind)
			continue
		}
		volume.Add(volume, transfer.value)
	}
	erc20Mutex.RUnlock()

	// removing old data, looping backwards to avoid issues with indicies
	for i := len(removing) - 1; i >= 0; i-- {
		erc20Mutex.Lock()
		erc20Transfers[token] = append(erc20Transfers[token][:i], erc20Transfers[token][i+1:]...)
		erc20Mutex.Unlock()
	}

	// set volume
	erc20Mutex.Lock()
	erc20Volume[token] = volume
	erc20Mutex.Unlock()
}

func ProcessERC20Transfer(p ActionEventData) {
	// validate decoded event have fields we're expecting
	if _, ok := p.DecodedTopics["from"]; !ok {
		return
	}
	if _, ok := p.DecodedTopics["to"]; !ok {
		return
	}
	if _, ok := p.DecodedData["value"]; !ok {
		return
	}

	from := p.DecodedTopics["from"].(common.Address)
	to := p.DecodedTopics["to"].(common.Address)
	value := p.DecodedData["value"].(*big.Int)

	tokenAddress := p.EventLog.Address
	tokenInfo := shared.RetrieveERC20Info(tokenAddress)

	blockNum := p.EventLog.BlockNumber
	block := shared.GetBlockHeader(big.NewInt(int64(blockNum)))
	timestamp := block.Time

	erc20 := ERC20TransferInfo{
		from:      from,
		to:        to,
		value:     value,
		decimals:  tokenInfo.Decimals,
		symbol:    tokenInfo.Symbol,
		timestamp: timestamp,
	}

	erc20Mutex.Lock()
	erc20Transfers[tokenAddress] = append(erc20Transfers[tokenAddress], erc20)
	erc20Mutex.Unlock()

	// send discord embed msg
	// SendWebhookMsgERC20Transfer(p.EventLog, erc20)

	// update token volume
	updateERC20Volume(tokenAddress)
}

// helper functions
