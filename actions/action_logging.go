package actions

import (
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"strconv"
	"strings"
	"time"

	discordwebhook "github.com/Zellic/EVM-trackooor/discord-webhook"
	"github.com/Zellic/EVM-trackooor/shared"
	"github.com/Zellic/EVM-trackooor/utils"

	"github.com/ethereum/go-ethereum/core/types"
)

// buffer for discord message to prevent rate limit
// events emitted will pile up in this buffer until it reaches
// length of 10, then it will send the whole buffer
// max 10 embeds, may start dropping events after limit reached
var embedBuffer []discordwebhook.Embed

var logToTerminal bool
var logToDiscord bool

var logEvents bool
var logTxs bool
var logBlocks bool

// logs deployment of any contract, not only contracts deployed by addresses we are tracking
var logAnyDeployments bool

// whether or not we want to know if tx was contract or EOA
// as it takes 1 RPC request per tx to do so
var determineTxType bool

var bufferDiscordMessages bool

func (actionInfo) InfoLogging() actionInfo {
	name := "Logging"
	overview := "Simply logs events/transactions from specified addresses, to terminal/Discord webhook. " +
		"Also contains option to log newly mined blocks."

	description := `This is useful for tracking activities done by specific addresses, or event signatures, and logging them.
Note that event ABI must be added using evm trackooor data command for it to be logged.`

	options := `"log-events" - bool, whether to log events from the specified addresses
"log-transactions" - bool, whether to log txs (including deployment txs) from the specified addresses
"log-blocks" - bool, whether to log newly mined blocks
"log-any-deployments" - bool, whether to log any deployments regardless of deployer address
"determine-tx-type" - bool, whether to determine the tx type (EOA to EOA, EOA to contract), requires additional RPC call, don't enable if lots of txs
"enable-terminal-logs" - bool, whether to log to the terminal
"enable-discord-logs" - bool, whether to log to Discord webhook
"discord-log-options" - dict, options for Discord webook logs
	"webhook-url" - Discord webook URL
	"username" - Discord webhook username (optional)
	"avatar-url" - Discord webhook avatar (optional)
	"buffer-webhook-messages" - bool, whether to buffer msgs (use if lots of event logs)
	"retry-webhook-messages" - bool, whether to retry failed msgs such as due to rate limit`

	example := `"Logging":{
    "addresses":{
        "0xdAC17F958D2ee523a2206206994597C13D831ec7":{}
    },
    "options":{
        "log-events": true,
        "log-transactions": true,
        "log-blocks": true,
        "log-any-deployments": false,
        "determine-tx-type": true,
        "enable-terminal-logs": true,
        "enable-discord-logs": false,
        "discord-log-options": {
            "webhook-url": "...",
            "username": "evm trackooor",
            "avatar-url": "...",
            "buffer-webhook-messages": true,
            "retry-webhook-messages": false
        }
    },
    "enabled":true
}`

	return actionInfo{
		ActionName:          name,
		ActionOverview:      overview,
		ActionDescription:   description,
		ActionOptionDetails: options,
		ActionConfigExample: example,
	}
}

func (p action) InitLogging() {

	if v, ok := p.o.CustomOptions["enable-terminal-logs"]; ok {
		logToTerminal = v.(bool)
	}
	if v, ok := p.o.CustomOptions["enable-discord-logs"]; ok {
		logToDiscord = v.(bool)
		// webhook settings
		if v, ok := p.o.CustomOptions["discord-log-options"]; ok {
			mapInterface := v.(map[string]interface{})
			setDiscordWebhookOptions(mapInterface)
		} else {
			log.Fatalf("Discord logs enabled \"discord-log-options\" was not found!")
		}
	}
	if v, ok := p.o.CustomOptions["log-events"]; ok {
		logEvents = v.(bool)
	}
	if v, ok := p.o.CustomOptions["log-transactions"]; ok {
		logTxs = v.(bool)
	}
	if v, ok := p.o.CustomOptions["log-blocks"]; ok {
		logBlocks = v.(bool)
	}
	if v, ok := p.o.CustomOptions["determine-tx-type"]; ok {
		determineTxType = v.(bool)
	}
	if v, ok := p.o.CustomOptions["log-any-deployments"]; ok {
		logAnyDeployments = v.(bool)
	}

	if logToTerminal || logToDiscord {
		if logEvents {
			// a separate map for events map address -> action will be better?
			for _, address := range p.o.Addresses {
				addAddressEventAction(address, logEvent)
			}
		}
		if logTxs {
			for _, address := range p.o.Addresses {
				addTxAddressAction(address, logTransaction)
			}
		}
		if logBlocks {
			addBlockAction(logBlockMined)
		}
		if logAnyDeployments { // we will need to get all txs in a block and check if tx is a deployment tx
			addBlockAction(logDeployments)
		}
	}
}

func setDiscordWebhookOptions(optionsMap map[string]interface{}) {
	// webhook url must be set
	if v, ok := optionsMap["webhook-url"]; ok {
		shared.DiscordWebhook.WebhookURL = v.(string)
	} else {
		log.Fatalf("Events logging: discord \"webhook-url\" not provided in \"discord-log-options\"!")
		return
	}
	if v, ok := optionsMap["username"]; ok {
		shared.DiscordWebhook.Username = v.(string)
	}
	if v, ok := optionsMap["avatar-url"]; ok {
		shared.DiscordWebhook.Avatar_url = v.(string)
	}
	if v, ok := optionsMap["buffer-webhook-messages"]; ok {
		bufferDiscordMessages = v.(bool)
	}
	if v, ok := optionsMap["retry-webhook-messages"]; ok {
		shared.DiscordWebhook.RetrySendingMessages = v.(bool)
	}
}

// EVENTS

func logEvent(p ActionEventData) {
	eventSigHash := p.EventLog.Topics[0]
	// event sig hash recognised
	if eventAbiInterface, ok := shared.EventSigs[eventSigHash.Hex()]; ok {
		eventAbi := eventAbiInterface.(map[string]interface{})
		output, discordOutput := utils.FormatEventInfo(p.EventFields, p.DecodedTopics, p.DecodedData)

		eventInfo := shared.EventInfo{
			Name: eventAbi["name"].(string),
			Sig:  eventAbi["sig"].(string),
		}

		// output to terminal
		if logToTerminal {
			fmt.Printf(
				"Event '%v' %v emitted from %v blocknum %v\n",
				eventInfo.Name,
				eventInfo.Sig,
				p.EventLog.Address,
				p.EventLog.BlockNumber,
			)
			fmt.Print(output)
		}

		// output to discord embed message
		if logToDiscord {
			logEventToDiscord(discordOutput, eventInfo, p.EventLog)
		}
	} else {
		// event sig hash not recognised
		if logToTerminal {
			fmt.Printf(
				"Unrecognised event emitted from %v, topics %v data %v blocknum %v\n",
				p.EventLog.Address,
				p.EventLog.Topics,
				"0x"+hex.EncodeToString(p.EventLog.Data),
				p.EventLog.BlockNumber,
			)
		}
	}
}

func logEventToDiscord(discordOutput string, eventInfo shared.EventInfo, vLog types.Log) {
	var discordEmbedMsg discordwebhook.Embed

	discordEmbedMsg.Title = eventInfo.Sig
	discordEmbedAuthor := discordwebhook.Author{
		Name: vLog.Address.Hex(),
	}
	fromUrlStr := utils.BlockscanFormat("address", vLog.Address.Hex())
	// if we managed to format the URL (otherwise embed will fail to send)
	if strings.HasPrefix(fromUrlStr, "http") {
		discordEmbedAuthor.Url = fromUrlStr
	}
	discordEmbedMsg.Author = discordEmbedAuthor

	TxUrlStr := utils.BlockscanFormat("transaction", vLog.TxHash.Hex())
	if strings.HasPrefix(TxUrlStr, "http") {
		discordEmbedMsg.Url = TxUrlStr
	}
	discordEmbedMsg.Description = discordOutput
	discordEmbedMsg.Timestamp = time.Now()

	// unique embed colour based on event signature hash
	eventSigHash := vLog.Topics[0]
	colourHex := eventSigHash.Hex()[2 : 2+6]
	colourNum, _ := strconv.ParseUint(colourHex, 16, 64)
	discordEmbedMsg.Color = int(colourNum)

	// if using buffers
	if bufferDiscordMessages {
		// fmt.Printf("embedBuffer length: %v\n", len(embedBuffer))
		// add embed to embed buffer
		embedBuffer = append(embedBuffer, discordEmbedMsg)
		if len(embedBuffer) > 10 {
			// cant have more than 10 embeds in a single message
			// split them up and send them in chunks
			for i := 0; i < len(embedBuffer); i += 10 {
				var err error
				if i+10 > len(embedBuffer) {
					err = shared.DiscordWebhook.SendEmbedMessages(embedBuffer[i:])
				} else {
					err = shared.DiscordWebhook.SendEmbedMessages(embedBuffer[i : i+10])
				}
				if err != nil {
					shared.Warnf(slog.Default(), "Webhook error: %v", err)
				}
			}
			embedBuffer = nil // clear buffer after sending them
		} else {
			err := shared.DiscordWebhook.SendEmbedMessages(embedBuffer)
			embedBuffer = nil
			if err != nil {
				shared.Warnf(slog.Default(), "Webhook error: %v", err)
			}
		}
	} else {
		err := shared.DiscordWebhook.SendEmbedMessages([]discordwebhook.Embed{discordEmbedMsg})
		if err != nil {
			shared.Warnf(slog.Default(), "Webhook error: %v", err)
		}
	}
}

// TRANSACTIONS

func logTransaction(p ActionTxData) {
	if logToTerminal {
		logTxToTerminal(p)
	}
	if logToDiscord {
		logTxToDiscord(p)
	}
}

func logTxToTerminal(p ActionTxData) {
	tx := p.Transaction
	block := p.Block
	to := p.To
	from := p.From
	if utils.IsDeploymentTx(tx) {
		deployedContract, _ := shared.GetDeployedContractAddress(tx)
		fmt.Printf(
			"Deployment TX by %v\n"+
				"contract: %v\n"+
				"value: %v wei (%v ether)\n"+
				"tx hash: %v\n",
			from,
			deployedContract,
			tx.Value(), utils.FormatDecimals(tx.Value(), 18),
			tx.Hash(),
		)
	} else {
		var txTypeStr string
		if determineTxType {
			txType := shared.DetermineTxType(tx, block.Number())
			txTypeStr = shared.TxTypes[txType] + " TX"
		} else {
			txTypeStr = "TX"
		}
		fmt.Printf(
			"%v\n"+
				"tx sender: %v\n"+
				"to: %v\n"+
				"value: %v wei (%v ether)\n"+
				"tx hash: %v\n",
			txTypeStr,
			from,
			to,
			tx.Value(), utils.FormatDecimals(tx.Value(), 18),
			tx.Hash(),
		)
	}
}

// logs to discord embed
func logTxToDiscord(p ActionTxData) {
	var discordEmbedMsg discordwebhook.Embed

	tx := p.Transaction
	block := p.Block
	to := p.To
	from := p.From

	// check if deployment tx
	if utils.IsDeploymentTx(tx) {
		deployedContract, _ := shared.GetDeployedContractAddress(tx)
		// embed title
		discordEmbedMsg.Title = "Deployment TX"
		// embed title link
		TxUrlStr := utils.BlockscanFormat("transaction", tx.Hash().Hex())
		if strings.HasPrefix(TxUrlStr, "http") {
			discordEmbedMsg.Url = TxUrlStr
		}
		// embed body
		discordEmbedMsg.Description = fmt.Sprintf(
			"tx sender: %v\ndeployed contract: %v\nvalue: %v wei (%v ether)",
			utils.FormatBlockscanHyperlink("address", from.Hex(), from.Hex()),
			utils.FormatBlockscanHyperlink("address", deployedContract.Hex(), deployedContract.Hex()),
			tx.Value(),
			utils.FormatDecimals(tx.Value(), 18),
		)
		// embed time
		discordEmbedMsg.Timestamp = time.Now()
		// unique embed colour based on tx sender (`from`)
		eventSigHash := from
		colourHex := eventSigHash.Hex()[2 : 2+6]
		colourNum, _ := strconv.ParseUint(colourHex, 16, 64)
		discordEmbedMsg.Color = int(colourNum)

		// send embed
		err := shared.DiscordWebhook.SendEmbedMessages([]discordwebhook.Embed{discordEmbedMsg})
		if err != nil {
			shared.Warnf(slog.Default(), "Webhook error: %v", err)
		}
	} else {
		// log normal tx

		// embed title
		var txTypeStr string
		if determineTxType {
			txType := shared.DetermineTxType(tx, block.Number())
			txTypeStr = shared.TxTypes[txType] + " TX"
		} else {
			txTypeStr = "TX"
		}
		discordEmbedMsg.Title = txTypeStr
		// embed title link
		TxUrlStr := utils.BlockscanFormat("transaction", tx.Hash().Hex())
		if strings.HasPrefix(TxUrlStr, "http") {
			discordEmbedMsg.Url = TxUrlStr
		}
		// embed body
		discordEmbedMsg.Description = fmt.Sprintf(
			"tx sender: %v\nto: %v\nvalue: %v wei (%v ether)",
			utils.FormatBlockscanHyperlink("address", from.Hex(), from.Hex()),
			utils.FormatBlockscanHyperlink("address", to.Hex(), to.Hex()),
			tx.Value(),
			utils.FormatDecimals(tx.Value(), 18),
		)
		// embed time
		discordEmbedMsg.Timestamp = time.Now()
		// unique embed colour based on tx sender (`from`)
		eventSigHash := from
		colourHex := eventSigHash.Hex()[2 : 2+6]
		colourNum, _ := strconv.ParseUint(colourHex, 16, 64)
		discordEmbedMsg.Color = int(colourNum)

		// send embed
		err := shared.DiscordWebhook.SendEmbedMessages([]discordwebhook.Embed{discordEmbedMsg})
		if err != nil {
			shared.Warnf(slog.Default(), "Webhook error: %v", err)
		}
	}
}

// BLOCKS

func logBlockMined(p ActionBlockData) {
	block := p.Block
	fmt.Printf("Block %v mined\n", block.Number())
	fmt.Printf("Transactions: %v\n", len(block.Transactions()))

	if logToDiscord {
		logBlockToDiscord(p)
	}
}

func logBlockToDiscord(p ActionBlockData) {
	var discordEmbedMsg discordwebhook.Embed

	discordEmbedMsg.Title = fmt.Sprintf("Block %v mined", p.Block.Number())
	// discordEmbedAuthor := discordwebhook.Author{}
	// fromUrlStr := utils.BlockscanFormat("address", vLog.Address.Hex())
	// // if we managed to format the URL (otherwise embed will fail to send)
	// if strings.HasPrefix(fromUrlStr, "http") {
	// 	discordEmbedAuthor.Url = fromUrlStr
	// }
	// discordEmbedMsg.Author = discordEmbedAuthor

	TxUrlStr := utils.BlockscanFormat("block", p.Block.Number().String())
	if strings.HasPrefix(TxUrlStr, "http") {
		discordEmbedMsg.Url = TxUrlStr
	}

	discordEmbedMsg.Description = fmt.Sprintf(
		"Transactions: %v\n"+
			"Time: %v\n",
		len(p.Block.Transactions()),
		p.Block.Time(),
	)
	discordEmbedMsg.Timestamp = time.Now()

	// unique embed colour based on block number
	colourHex := fmt.Sprintf("%06v", p.Block.Number().Text(16)[2:])[:6]
	colourNum, _ := strconv.ParseUint(colourHex, 16, 64)
	discordEmbedMsg.Color = int(colourNum)

	// send embed
	err := shared.DiscordWebhook.SendEmbedMessages([]discordwebhook.Embed{discordEmbedMsg})
	if err != nil {
		shared.Warnf(slog.Default(), "Webhook error: %v", err)
	}
}

func logDeployments(p ActionBlockData) {
	block := p.Block
	for _, tx := range block.Transactions() {
		// To() returns nil if tx is deployment tx
		if tx.To() == nil {
			to := tx.To()
			from := utils.GetTxSender(tx)
			txData := ActionTxData{
				Transaction: tx,
				Block:       block,
				To:          to,
				From:        from,
			}
			if logToTerminal {
				logTxToTerminal(txData)
			}
			if logToDiscord {
				logTxToDiscord(txData)
			}
		}
	}
}
