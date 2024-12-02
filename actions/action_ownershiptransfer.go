package actions

import (
	"evm-trackooor/shared"
	"evm-trackooor/utils"
	"fmt"
	"log"
	"math/big"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

// logging
var ownershipTransferLog *log.Logger

var thresholdUSD float64

// discord webhook
var ownershipTransferWebhook shared.WebhookInstance

var abr *utils.AddressBalanceRetriever

func (actionInfo) InfoOwnershipTransferTrack() actionInfo {
	name := "OwnershipTransferTrack"
	overview := `Listens to OwnershipTransferred events, ` +
		`checks contract balance (ETH + specified ERC20s) of contract that emitted the event, ` +
		`and logs via discord webhook if the contract balance surpasses a certain threshold`

	description := `Four variations of similar func sigs are tracked:
OwnershipTransferred(address,address)
AdminChanged(address,address)
GovernorTransferred(address,address)
RoleGranted(bytes32,address,address)

Contract's USD balance is determined by aggregating USD balance of ether and specified ERC20 tokens.
Price of ether and tokens are retrieved using Coin API, which may not support prices for all ERC20 tokens.`

	options := `"threshold" - USD threshold, contract balance above this will cause a discord alert
"coin-api-key" - Coin API key from https://www.coinapi.io/
"webhook-url" - URL of discord webhook to send discord alerts
"tokens" - array of ERC20 token addresses, to take into account when determining contract USD balance`

	example := `"OwnershipTransferTrack": {
	"addresses":{},
	"options":{
		"threshold":10000,
		"webhook-url":"https://discord.com/api/webhooks/...",
		"coin-api-key":"XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX",
		"tokens":["0xdac17f958d2ee523a2206206994597c13d831ec7",
		"0xB8c77482e45F1F44dE1745F52C74426C631bDD52",
		...]
	}
}`

	return actionInfo{
		ActionName:          name,
		ActionOverview:      overview,
		ActionDescription:   description,
		ActionOptionDetails: options,
		ActionConfigExample: example,
	}
}

func (p action) InitOwnershipTransferTrack() {
	// logging
	ownershipTransferLog = shared.TimeLogger("[OwnershipTransferTrack] ")

	// set threshold
	var ok bool
	thresholdUSD, ok = p.o.CustomOptions["threshold"].(float64)
	if !ok {
		ownershipTransferLog.Fatalf("Please specify USD threshold \"threshold\"!")
	}
	ownershipTransferLog.Printf("Set threshold to $%v USD\n", thresholdUSD)

	// discord webhook
	webhookURL, ok := p.o.CustomOptions["webhook-url"].(string)
	if !ok {
		ownershipTransferLog.Fatalf("Please specify \"webhook-url\"!")
	}
	ownershipTransferWebhook = shared.WebhookInstance{
		WebhookURL:           webhookURL,
		Username:             "ownership transfer track",
		Avatar_url:           "",
		RetrySendingMessages: true,
	}

	// coin api key
	coinApiKey, ok := p.o.CustomOptions["coin-api-key"].(string)
	if !ok {
		ownershipTransferLog.Fatalf("Please specify \"coin-api-key\"!")
	}

	// address balance retriever to get USD value of address
	var tokens []common.Address
	for _, tokenAddrHex := range p.o.CustomOptions["tokens"].([]interface{}) {
		tokenAddress := common.HexToAddress(tokenAddrHex.(string))
		tokens = append(tokens, tokenAddress)
	}

	abr = utils.NewAddressBalanceRetriever(coinApiKey, tokens)

	/*

		OwnershipTransferred(address indexed previousOwner, address indexed newOwner)
		AdminChanged(address indexed previousAdmin, address indexed newAdmin)
		GovernorTransferred(address indexed previousGovernor, address indexed newGovernor)
		RoleGranted(bytes32 indexed role, address indexed account, address indexed sender)

	*/
	addEventSigAction("OwnershipTransferred(address,address)", handleOwnershipTransferred)
	addEventSigAction("AdminChanged(address,address)", handleAdminChanged)
	addEventSigAction("GovernorTransferred(address,address)", handleGovernorTransferred)
	addEventSigAction("RoleGranted(bytes32,address,address)", handleRoleGranted)

}

func handleOwnershipTransferred(p ActionEventData) {
	// validate decoded topics contains what we're expecting
	_, ok := p.DecodedTopics["previousOwner"]
	if !ok {
		return
	}
	_, ok = p.DecodedTopics["newOwner"]
	if !ok {
		return
	}

	previousOwner := p.DecodedTopics["previousOwner"].(common.Address)
	newOwner := p.DecodedTopics["newOwner"].(common.Address)

	// ignore if previously no owner (e.g contract init)
	if previousOwner.Cmp(shared.ZeroAddress) == 0 {
		fmt.Printf("zero addr, ignoring - previousOwner: %v\n", previousOwner)
		return
	}

	contract := p.EventLog.Address
	txHash := p.EventLog.TxHash
	blockNum := big.NewInt(int64(p.EventLog.BlockNumber))

	contractBalanceUSD := abr.GetAddressUSDbalance(contract, blockNum)

	// ownershipTransferLog.Printf("contract: %v txhash: %v previousOwner: %v newOwner: %v\n", contract, txHash, previousOwner, newOwner)
	// ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

	if contractBalanceUSD > thresholdUSD {
		ownershipTransferLog.Printf("sending discord msg OwnershipTransferred\n")
		ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

		ownershipTransferWebhook.SendMessage(fmt.Sprintf(`## OwnershipTransferred event
%v %v
previousOwner: %v
newOwner: %v
balance (USD): %v`,
			utils.FormatBlockscanHyperlink("address", "`contract`", contract.Hex()),
			utils.FormatBlockscanHyperlink("transaction", "`tx hash`", txHash.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(previousOwner), previousOwner.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(newOwner), newOwner.Hex()),
			utils.CodeQuote(strconv.FormatFloat(contractBalanceUSD, 'f', -1, 64))),
		)
		ownershipTransferLog.Printf("discord msg sent\n")
	}
}

func handleAdminChanged(p ActionEventData) {
	// validate decoded topics contains what we're expecting
	_, ok := p.DecodedTopics["previousAdmin"]
	if !ok {
		return
	}
	_, ok = p.DecodedTopics["newAdmin"]
	if !ok {
		return
	}

	previousAdmin := p.DecodedTopics["previousAdmin"].(common.Address)
	newAdmin := p.DecodedTopics["newAdmin"].(common.Address)

	// ignore if previously no owner (e.g contract init)
	if previousAdmin.Cmp(shared.ZeroAddress) == 0 {
		fmt.Printf("zero addr, ignoring - previousAdmin: %v\n", previousAdmin)
		return
	}

	contract := p.EventLog.Address
	txHash := p.EventLog.TxHash
	blockNum := big.NewInt(int64(p.EventLog.BlockNumber))

	contractBalanceUSD := abr.GetAddressUSDbalance(contract, blockNum)

	// ownershipTransferLog.Printf("contract: %v txhash: %v previousOwner: %v newOwner: %v\n", contract, txHash, previousOwner, newOwner)
	// ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

	if contractBalanceUSD > thresholdUSD {
		ownershipTransferLog.Printf("sending discord msg AdminChanged\n")
		ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

		ownershipTransferWebhook.SendMessage(fmt.Sprintf(`## AdminChanged event
%v %v
previousAdmin: %v
newAdmin: %v
balance (USD): %v`,
			utils.FormatBlockscanHyperlink("address", "`contract`", contract.Hex()),
			utils.FormatBlockscanHyperlink("transaction", "`tx hash`", txHash.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(previousAdmin), previousAdmin.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(newAdmin), newAdmin.Hex()),
			utils.CodeQuote(strconv.FormatFloat(contractBalanceUSD, 'f', -1, 64))),
		)
		ownershipTransferLog.Printf("discord msg sent\n")
	}
}

func handleGovernorTransferred(p ActionEventData) {
	// validate decoded topics contains what we're expecting
	_, ok := p.DecodedTopics["previousGovernor"]
	if !ok {
		return
	}
	_, ok = p.DecodedTopics["newGovernor"]
	if !ok {
		return
	}

	previousGovernor := p.DecodedTopics["previousGovernor"].(common.Address)
	newGovernor := p.DecodedTopics["newGovernor"].(common.Address)

	// ignore if previously no owner (e.g contract init)
	if previousGovernor.Cmp(shared.ZeroAddress) == 0 {
		fmt.Printf("zero addr, ignoring - previousGovernor: %v\n", previousGovernor)
		return
	}

	contract := p.EventLog.Address
	txHash := p.EventLog.TxHash
	blockNum := big.NewInt(int64(p.EventLog.BlockNumber))

	contractBalanceUSD := abr.GetAddressUSDbalance(contract, blockNum)

	// ownershipTransferLog.Printf("contract: %v txhash: %v previousOwner: %v newOwner: %v\n", contract, txHash, previousOwner, newOwner)
	// ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

	if contractBalanceUSD > thresholdUSD {
		ownershipTransferLog.Printf("sending discord msg GovernorTransferred\n")
		ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

		ownershipTransferWebhook.SendMessage(fmt.Sprintf(`## GovernorTransferred event
%v %v
previousGovernor: %v
newGovernor: %v
balance (USD): %v`,
			utils.FormatBlockscanHyperlink("address", "`contract`", contract.Hex()),
			utils.FormatBlockscanHyperlink("transaction", "`tx hash`", txHash.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(previousGovernor), previousGovernor.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(newGovernor), newGovernor.Hex()),
			utils.CodeQuote(strconv.FormatFloat(contractBalanceUSD, 'f', -1, 64))),
		)
		ownershipTransferLog.Printf("discord msg sent\n")
	}
}

func handleRoleGranted(p ActionEventData) {
	// validate decoded topics contains what we're expecting
	_, ok := p.DecodedTopics["previousGovernor"]
	if !ok {
		return
	}
	_, ok = p.DecodedTopics["newGovernor"]
	if !ok {
		return
	}
	_, ok = p.DecodedTopics["role"]
	if !ok {
		return
	}

	role := p.DecodedTopics["role"].([32]byte)
	account := p.DecodedTopics["account"].(common.Address)
	sender := p.DecodedTopics["sender"].(common.Address)

	contract := p.EventLog.Address
	txHash := p.EventLog.TxHash
	blockNum := big.NewInt(int64(p.EventLog.BlockNumber))

	contractBalanceUSD := abr.GetAddressUSDbalance(contract, blockNum)

	// ownershipTransferLog.Printf("contract: %v txhash: %v previousOwner: %v newOwner: %v\n", contract, txHash, previousOwner, newOwner)
	// ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

	if contractBalanceUSD > thresholdUSD {
		ownershipTransferLog.Printf("sending discord msg RoleGranted\n")
		ownershipTransferLog.Printf("contractBalanceUSD: %v\n", contractBalanceUSD)

		ownershipTransferWebhook.SendMessage(fmt.Sprintf(`## RoleGranted event
%v %v
account: %v
sender: %v
role: %v
balance (USD): %v`,
			utils.FormatBlockscanHyperlink("address", "`contract`", contract.Hex()),
			utils.FormatBlockscanHyperlink("transaction", "`tx hash`", txHash.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(account), account.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(sender), sender.Hex()),
			utils.CodeQuote(common.Bytes2Hex(role[:])),
			utils.CodeQuote(strconv.FormatFloat(contractBalanceUSD, 'f', -1, 64))),
		)
		ownershipTransferLog.Printf("discord msg sent\n")
	}
}
