package actions

import (
	"context"
	"evm-trackooor/shared"
	"evm-trackooor/utils"
	"fmt"
	"log"
	"strconv"

	"github.com/ethereum/go-ethereum/common"
)

var proxyUpgradesConfig struct {
	thresholdUSD float64
	webhook      shared.WebhookInstance
	coinApiKey   string
}

var proxyUpgradesLog *log.Logger

func (actionInfo) InfoProxyUpgrades() actionInfo {
	name := "ProxyUpgrades"
	overview := `Listens to proxy upgrade events, ` +
		`checks contract balance (ETH + specified ERC20s) of contract that emitted the event, ` +
		`and logs via discord webhook if the contract balance surpasses a certain threshold`

	description := `The event signature Upgraded(address) is monitored.

Contract's USD balance is determined by aggregating USD balance of ether and specified ERC20 tokens.
Price of ether and tokens are retrieved using Coin API, which may not support prices for all ERC20 tokens.`

	options := `"threshold" - USD threshold, contract balance above this will cause a discord alert
"coin-api-key" - Coin API key from https://www.coinapi.io/
"webhook-url" - URL of discord webhook to send discord alerts
"tokens" - array of ERC20 token addresses, to take into account when determining contract USD balance`

	example := `"ProxyUpgrades": {
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

func (p action) InitProxyUpgrades() {
	// logging
	proxyUpgradesLog = shared.TimeLogger("[Proxy Upgrades] ")

	// load config
	loadProxyUpgradesConfig(p.o.CustomOptions)

	// address balance retriever to get USD value of address
	var tokens []common.Address
	for _, tokenAddrHex := range p.o.CustomOptions["tokens"].([]interface{}) {
		tokenAddress := common.HexToAddress(tokenAddrHex.(string))
		tokens = append(tokens, tokenAddress)
	}

	abr = utils.NewAddressBalanceRetriever(proxyUpgradesConfig.coinApiKey, tokens)

	// track Upgraded(address indexed implementation)
	// this is emitted for both Transparent and UPPS proxies
	// (at least when using OpenZepplin's code)
	addEventSigAction("Upgraded(address)", handleProxyUpgrade)
}

func loadProxyUpgradesConfig(customOptions map[string]interface{}) {
	if v, ok := customOptions["coin-api-key"]; ok {
		proxyUpgradesConfig.coinApiKey = v.(string)
	} else {
		proxyUpgradesLog.Fatalf("Please specify \"coin-api-key\"!")
	}

	if v, ok := customOptions["threshold"]; ok {
		proxyUpgradesConfig.thresholdUSD = v.(float64)
	} else {
		proxyUpgradesLog.Fatalf("Please specify USD threshold \"threshold\"!")
	}

	if v, ok := customOptions["webhook-url"]; ok {
		proxyUpgradesConfig.webhook = shared.WebhookInstance{
			WebhookURL:           v.(string),
			Username:             "proxy upgrade track",
			Avatar_url:           "",
			RetrySendingMessages: true,
		}
	} else {
		proxyUpgradesLog.Fatalf("Please specify discord webhook URL \"webhook-url\"!")

	}

	proxyUpgradesLog.Printf("Set threshold to $%v USD\n", proxyUpgradesConfig.thresholdUSD)
}

func handleProxyUpgrade(p ActionEventData) {
	// validate decoded event have fields we're expecting
	if _, ok := p.DecodedTopics["implementation"]; !ok {
		return
	}

	proxy := p.EventLog.Address
	upgradedTo := p.DecodedTopics["implementation"].(common.Address)
	fmt.Printf("proxy %v upgraded to %v\n", proxy, upgradedTo)

	header, err := shared.Client.HeaderByHash(context.Background(), p.EventLog.BlockHash)
	if err != nil {
		proxyUpgradesLog.Printf("err when getting header: %v\n", err)
	}
	blockNum := header.Number

	contractUSDbalance := abr.GetAddressUSDbalance(proxy, blockNum)
	fmt.Printf("proxy usd bal: %v\n", contractUSDbalance)
	if contractUSDbalance >= proxyUpgradesConfig.thresholdUSD {
		fmt.Printf("proxy %v exceeds threshold usd, usd: %v upgraded to %v\n", proxy, contractUSDbalance, upgradedTo)
		// send discord webhook msg
		proxyUpgradesConfig.webhook.SendMessage(fmt.Sprintf(`## Proxy Upgraded
proxy: %v
upgradedTo: %v
proxy USD balance: %v`,
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(proxy), proxy.Hex()),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(upgradedTo), upgradedTo.Hex()),
			utils.CodeQuote(strconv.FormatFloat(contractUSDbalance, 'f', -1, 64)),
		))
	}
}
