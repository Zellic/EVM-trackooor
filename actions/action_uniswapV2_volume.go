package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/big"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/Zellic/EVM-trackooor/shared"
	"github.com/Zellic/EVM-trackooor/utils"

	"github.com/ethereum/go-ethereum/common"
)

// v2 pair structs

type v2Swap struct {
	// event data
	Sender     common.Address `json:"sender"`
	Amount0In  *big.Int       `json:"amount0In"`
	Amount1In  *big.Int       `json:"amount1In"`
	Amount0Out *big.Int       `json:"amount0Out"`
	Amount1Out *big.Int       `json:"amount1Out"`
	To         common.Address `json:"to"`
	// other data
	Timestamp time.Time `json:"timestamp"`
}

type v2PairInfo struct {
	Token0                 common.Address `json:"token0"`                    // token0 address
	Token1                 common.Address `json:"token1"`                    // token1 address
	Token0Volume           *big.Int       `json:"token0-volume"`             // volume of token0
	Token1Volume           *big.Int       `json:"token1-volume"`             // volume of token1
	Token0PrevPeriodVolume *big.Int       `json:"token0-prev-period-volume"` // volume of token0 in the prev X hour period
	Token1PrevPeriodVolume *big.Int       `json:"token1-prev-period-volume"` // volume of token1 in the prev X hour period

	LastAlerted time.Time `json:"last-alerted"` // last time webhook msg was sent

	Swaps []v2Swap `json:"swaps"` // swaps in the last <time duration>
}

// logging
var uniswapV2Log *log.Logger

// discord webhook
var v2DiscordWebhook shared.WebhookInstance

// custom options
var uniswapV2Options struct {
	v2PriceStaleDuration        time.Duration // max time until fetch new token price
	v2VolumeDuration            time.Duration // track volume in last <time duration>
	v2VolumeThresholdUSD        *big.Float
	v2VolumePercentageThreshold *big.Float // alert if volume percentage change from last X hours > threshold
	v2VolumeAlertTimeout        time.Duration
	v2WebhookUrl                string
	uniswapV2DataFilepath       string
}

// map v2 pair address to its info
var trackedV2Pairs map[common.Address]v2PairInfo
var trackedV2PairsMutex sync.RWMutex

func (p action) InitUniswapV2Volume() {
	// init maps
	trackedV2Pairs = make(map[common.Address]v2PairInfo)
	tokens = make(map[common.Address]tokenData) // from uniswapV3 volume (incase only running v2)

	// init logging
	uniswapV2Log = shared.TimeLogger("[Uniswap V2] ")

	// listen to Uniswap V2 pair Factory(s) for PairCreated events
	for _, address := range p.o.Addresses {
		addAddressEventSigAction(
			address,
			"PairCreated(address,address,address,uint256)",
			handleV2PairCreated,
		)
	}

	uniswapV2GetCustomOptions(p.o.CustomOptions)
}

func uniswapV2GetCustomOptions(CustomOptions map[string]interface{}) {
	// get data filepath
	if v, ok := CustomOptions["data-filepath"]; ok {
		uniswapV2Options.uniswapV2DataFilepath = v.(string)
	} else {
		uniswapV2Log.Fatal("\"data-filepath\" not specified!")
	}
	uniswapV2Log.Printf("using data file %v\n", uniswapV2Options.uniswapV2DataFilepath)

	// get api key
	if v, ok := CustomOptions["moralis-api-key"]; ok {
		moralisApiKey = v.(string)
	} else {
		uniswapV2Log.Fatalf(`Please provide API KEY "moralis-api-key"`)
	}

	if v, ok := CustomOptions["moralis-chain-string"]; ok {
		moralisChainString = v.(string)
	} else {
		uniswapV2Log.Fatalf(`Please provide chain "moralis-chain-string" (e.g. "eth")`)
	}

	if v, ok := CustomOptions["volume-threshold-usd"]; ok {
		s := v.(string)
		uniswapV2Options.v2VolumeThresholdUSD, ok = big.NewFloat(0).SetString(s)
		if !ok {
			uniswapV2Log.Fatalf("Unable to set volumeThresholdUSD with value: %v\n", s)
		}
	} else {
		uniswapV2Log.Fatalf("\"volume-threshold-usd\" not specified!")
	}
	uniswapV2Log.Printf("Using volume threshold USD %v\n", uniswapV2Options.v2VolumeThresholdUSD.Text('f', 0))

	// get discord webhook url
	if v, ok := CustomOptions["webhook-url"]; ok {
		uniswapV2Options.v2WebhookUrl = v.(string)
		v2DiscordWebhook = shared.WebhookInstance{
			WebhookURL:           uniswapV2Options.v2WebhookUrl,
			Username:             "evm trackooor",
			Avatar_url:           "",
			RetrySendingMessages: true,
		}
	} else {
		uniswapV2Log.Fatalf("\"webhook-url\" not specified!")
	}
	uniswapV2Log.Printf("Using discord webhook URL %v\n", uniswapV2Options.v2WebhookUrl)

	// get volume duration
	if v, ok := CustomOptions["volume-duration"]; ok {
		s := v.(string)
		duration, err := time.ParseDuration(s)
		if err != nil {
			uniswapV2Log.Fatalf("Invalid time duration %v\n", s)
		}
		uniswapV2Options.v2VolumeDuration = duration
		uniswapV2Log.Printf("Using volume duration of %v\n", uniswapV2Options.v2VolumeDuration.String())
	} else {
		uniswapV2Options.v2VolumeDuration = time.Hour * 24
		uniswapV2Log.Printf("\"volume-duration\" not set, defaulting to %v\n", uniswapV2Options.v2VolumeDuration)
	}

	// get volume alert timeout
	if v, ok := CustomOptions["volume-alert-timeout"]; ok {
		s := v.(string)
		duration, err := time.ParseDuration(s)
		if err != nil {
			uniswapV2Log.Fatalf("Invalid time duration %v\n", s)
		}
		uniswapV2Options.v2VolumeAlertTimeout = duration
		uniswapV2Log.Printf("Using volume alert timeout of %v\n", uniswapV2Options.v2VolumeAlertTimeout.String())
	} else {
		uniswapV2Options.v2VolumeAlertTimeout = time.Hour * 1
		uniswapV2Log.Printf("\"volume-alert-timeout\" not set, defaulting to %v\n", uniswapV2Options.v2VolumeAlertTimeout)
	}

	// get price stale duration
	if v, ok := CustomOptions["price-stale-duration"]; ok {
		s := v.(string)
		duration, err := time.ParseDuration(s)
		if err != nil {
			uniswapV2Log.Fatalf("Invalid time duration %v\n", s)
		}
		uniswapV2Options.v2PriceStaleDuration = duration
		uniswapV2Log.Printf("Using price stale duration of %v\n", uniswapV2Options.v2PriceStaleDuration.String())
	} else {
		uniswapV2Options.v2PriceStaleDuration = time.Minute * 10
		uniswapV2Log.Printf("\"price-stale-duration\" not set, defaulting to %v\n", uniswapV2Options.v2PriceStaleDuration)
	}

	// get volume percentage change threshold
	if v, ok := CustomOptions["volume-percentage-threshold"]; ok {
		s := v.(string)
		uniswapV2Options.v2VolumePercentageThreshold, ok = big.NewFloat(0).SetString(s)
		if !ok {
			uniswapV2Log.Fatalf("Unable to set volumeThresholdUSD with value: %v\n", s)
		}
		uniswapV2Log.Printf("Using percentage change threshold of %v\n", uniswapV2Options.v2VolumePercentageThreshold.String())
	} else {
		uniswapV2Options.v2VolumePercentageThreshold = big.NewFloat(200)
		uniswapV2Log.Printf("\"volume-percentage-threshold\" not set, defaulting to %v\n", uniswapV2Options.v2VolumePercentageThreshold)
	}

	// set token addresses
	// USDT
	if v, ok := CustomOptions["usdt-address"]; ok {
		USDT = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		USDT = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	} else {
		uniswapV2Log.Fatalf("\"usdt-address\" not provided! Please provide USDT token address.")
	}

	// USDC
	if v, ok := CustomOptions["usdc-address"]; ok {
		USDC = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		USDC = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	} else {
		uniswapV2Log.Fatalf("\"usdc-address\" not provided! Please provide USDC token address.")
	}
	// WETH
	if v, ok := CustomOptions["weth-address"]; ok {
		WETH = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		WETH = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	} else {
		uniswapV2Log.Fatalf("\"weth-address\" not provided! Please provide WETH token address.")
	}

	// get WETH/USDC uniswap v3 pool address
	// v3 pool used as this is just for price oracle
	if v, ok := CustomOptions["weth-usdc-pool"]; ok {
		wethUsdcV3Pool = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		wethUsdcV3Pool = common.HexToAddress("0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640")
	} else {
		uniswapV2Log.Fatalf("\"weth-usdc-pool\" not provided! Please provide WETH/USDc uniswap V3 pool address.")
	}

	// load previously saved data file (if exists)
	if _, err := os.Stat(uniswapV2Options.uniswapV2DataFilepath); err == nil {
		// data file exists, load previous data
		dat, err := os.ReadFile(uniswapV2Options.uniswapV2DataFilepath)
		if err != nil {
			panic(err)
		}

		// load JSON data
		trackedV2PairsMutex.Lock()
		err = json.Unmarshal(dat, &trackedV2Pairs)
		if err != nil {
			panic(err)
		}

		uniswapV2Log.Printf("Loaded saved data, tracking %v pairs\n", len(trackedV2Pairs))
		// add previous tracked pairs to tracking
		for pair := range trackedV2Pairs {
			addAddressEventSigAction(pair, "Swap(address,address,int256,int256,uint160,uint128,int24)", handleV2PairSwapEvent)
		}
		trackedV2PairsMutex.Unlock()

	} else if errors.Is(err, os.ErrNotExist) {
		// data file does not exist, all good
	} else {
		panic(err)
	}
}

func handleV2PairCreated(p ActionEventData) {
	token0 := p.DecodedTopics["token0"].(common.Address)
	token1 := p.DecodedTopics["token1"].(common.Address)
	pair := p.DecodedData["pair"].(common.Address)

	// track the pair for swap events
	addAddressEventSigAction(
		pair,
		"Swap(address,uint256,uint256,uint256,uint256,address)",
		handleV2PairSwapEvent,
	)

	// fmt.Printf("New uniswapv2 pair created: %v\n", pair)

	// add pair to map
	trackedV2PairsMutex.Lock()
	trackedV2Pairs[pair] = v2PairInfo{
		Token0: token0,
		Token1: token1,
	}
	trackedV2PairsMutex.Unlock()

	// update datafile
	uniswapV2UpdateDataFile()
}

func handleV2PairSwapEvent(p ActionEventData) {
	pair := p.EventLog.Address

	swapData := v2Swap{
		Sender:     p.DecodedTopics["sender"].(common.Address),
		Amount0In:  p.DecodedData["amount0In"].(*big.Int),
		Amount1In:  p.DecodedData["amount1In"].(*big.Int),
		Amount0Out: p.DecodedData["amount0Out"].(*big.Int),
		Amount1Out: p.DecodedData["amount1Out"].(*big.Int),
		To:         p.DecodedTopics["to"].(common.Address),
	}

	// fmt.Printf("swapData: %v\n", swapData)

	// append swap data
	trackedV2PairsMutex.Lock()
	pairData := trackedV2Pairs[pair]
	pairData.Swaps = append(pairData.Swaps, swapData)
	trackedV2Pairs[pair] = pairData
	trackedV2PairsMutex.Unlock()

	// update pair volume
	updateV2Volume(pair)

	// check pair volume and alert if needed
	checkV2Volume(pair)

	// update datafile
	uniswapV2UpdateDataFile()
}

func updateV2Volume(pair common.Address) {
	// loop through all swaps
	trackedV2PairsMutex.RLock()
	pairData := trackedV2Pairs[pair]
	trackedV2PairsMutex.RUnlock()

	swaps := pairData.Swaps
	token0Volume := big.NewInt(0)
	token1Volume := big.NewInt(0)
	token0PrevVolume := big.NewInt(0)
	token1PrevVolume := big.NewInt(0)

	now := time.Now()
	removing := []int{} // swap indexes to remove due to being old
	for ind, swap := range swaps {
		// check if swap within timeframe
		swapTime := swap.Timestamp
		// time within current period (swapTime + volumeDuration > now)
		if swapTime.Add(uniswapV2Options.v2VolumeDuration).After(now) {
			// add amounts of token0 traded
			// token0Volume += Amount0In + Amount0Out
			token0Volume.Add(token0Volume, big.NewInt(0).Add(swap.Amount0In, swap.Amount0Out))
			// add amounts of token1 traded
			// token1Volume += Amount1In + Amount1Out
			token1Volume.Add(token1Volume, big.NewInt(0).Add(swap.Amount1In, swap.Amount1Out))
		} else if swapTime.Add(uniswapV2Options.v2VolumeDuration * 2).After(now) {
			// time within previous period (swapTime + volumeDuration*2 > now)
			token0PrevVolume.Add(token0PrevVolume, big.NewInt(0).Add(swap.Amount0In, swap.Amount0Out))
			token1PrevVolume.Add(token1PrevVolume, big.NewInt(0).Add(swap.Amount1In, swap.Amount1Out))
		} else {
			// swap too old, remove
			removing = append(removing, ind)
		}
	}

	// remove old swap data, looping backwards through removing to avoid index issues
	for i := len(removing) - 1; i >= 0; i-- {
		ind := removing[i]
		swaps = append(swaps[:ind], swaps[ind+1:]...)
	}

	// set volume and swap data
	pairData.Token0Volume = token0Volume
	pairData.Token1Volume = token1Volume
	pairData.Token0PrevPeriodVolume = token0PrevVolume
	pairData.Token1PrevPeriodVolume = token1PrevVolume
	pairData.Swaps = swaps

	// assign to map
	trackedV2PairsMutex.Lock()
	trackedV2Pairs[pair] = pairData
	trackedV2PairsMutex.Unlock()
}

func checkV2Volume(pair common.Address) {
	trackedV2PairsMutex.RLock()
	pairData := trackedV2Pairs[pair]
	trackedV2PairsMutex.RUnlock()

	// if either token is a common token, we will use that token for price and end there
	// check token0
	if slices.Contains([]common.Address{USDT, USDC, WETH}, pairData.Token0) {
		updateTokenInfo(pairData.Token0)
		token0Alerted := alertV2Volume(pair, pairData.Token0, pairData.Token0Volume, pairData.Token0PrevPeriodVolume)
		if token0Alerted {
			uniswapV2Log.Printf("V2 Pair %v alerted for token0 (shortcut price)\n", pair)
		}
		return
	}
	// check token1
	if slices.Contains([]common.Address{USDT, USDC, WETH}, pairData.Token1) {
		updateTokenInfo(pairData.Token1)
		token1Alerted := alertV2Volume(pair, pairData.Token1, pairData.Token1Volume, pairData.Token1PrevPeriodVolume)
		if token1Alerted {
			uniswapV2Log.Printf("V2 Pair %v alerted for token1 (shortcut price)\n", pair)
		}
		return
	}

	// otherwise we will just fetch prices as usual

	// update token prices
	updateTokenInfo(pairData.Token0)
	updateTokenInfo(pairData.Token1)

	// send alerts to webhook (if volume > threshold)
	// token0
	token0Alerted := alertV2Volume(pair, pairData.Token0, pairData.Token0Volume, pairData.Token0PrevPeriodVolume)
	if token0Alerted {
		uniswapV2Log.Printf("V2 Pair %v alerted for token0\n", pair)
		return
	}
	// token1
	token1Alerted := alertV2Volume(pair, pairData.Token1, pairData.Token1Volume, pairData.Token1PrevPeriodVolume)
	if token1Alerted {
		uniswapV2Log.Printf("V2 Pair %v alerted for token1\n", pair)
		return
	}
}

// alert to webhook (if volume > threshold), returns whether or not alert was sent
func alertV2Volume(pair common.Address, tokenAddress common.Address, tokenVolume *big.Int, tokenPrevVolume *big.Int) bool {
	trackedV2PairsMutex.RLock()
	pairData := trackedV2Pairs[pair]
	trackedV2PairsMutex.RUnlock()

	tokensMutex.RLock()
	token := tokens[tokenAddress]
	token0 := tokens[pairData.Token0]
	token1 := tokens[pairData.Token1]
	tokensMutex.RUnlock()

	// don't alert if token info (e.g. price) doesnt exist
	if token.Info.TokenAddress == "" {
		return false
	}

	now := time.Now()

	// get volumes in USD
	tokenVolumeUSD := calculateUSDpriceOfTokens(tokenAddress, tokenVolume)
	tokenPrevVolumeUSD := calculateUSDpriceOfTokens(tokenAddress, tokenPrevVolume)

	//DEBUG
	fmt.Printf("token: %v\n", token.Info.TokenName)
	fmt.Printf("tokenVolumeUSD: %v\n", tokenVolumeUSD)
	fmt.Printf("tokenPrevVolumeUSD: %v\n", tokenPrevVolumeUSD)

	// if we already alerted before, then we are looking for percentage change
	// otherwise we're looking for USD volume > threshold
	// have we alerted before?
	if pairData.LastAlerted.IsZero() {
		// we have not alerted before, alert if exceed volume threshold
		if tokenVolumeUSD.Cmp(uniswapV2Options.v2VolumeThresholdUSD) < 0 {
			// did not exceed threshold
			return false
		}
		// send webhook msg
		v2DiscordWebhook.SendMessage(fmt.Sprintf(
			"## Uniswap V2 Pair - Volume alert\n"+
				"Pair: %v %v\n"+
				"Token0 %v (%v): %v %v\n"+
				"Token1 %v (%v): %v %v\n"+
				"Volume of %v (%v) in last %v: %v USD",
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pair), pair.Hex()),
			formatUniswapHyperlink("pair", "uniswap", pair.Hex()),
			utils.CodeQuote(token0.Info.TokenName), utils.CodeQuote(token0.Info.TokenSymbol),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pairData.Token0), pairData.Token0.Hex()),
			formatUniswapHyperlink("token", "uniswap", pairData.Token0.Hex()),
			utils.CodeQuote(token1.Info.TokenName), utils.CodeQuote(token1.Info.TokenSymbol),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pairData.Token1), pairData.Token1.Hex()),
			formatUniswapHyperlink("token", "uniswap", pairData.Token1.Hex()),
			utils.CodeQuote(token.Info.TokenName), utils.CodeQuote(token.Info.TokenSymbol), uniswapV2Options.v2VolumeDuration,
			utils.CodeQuote(tokenVolumeUSD.Text('f', 2)),
		))

		// assign to map
		trackedV2PairsMutex.Lock()
		trackedV2Pairs[pair] = pairData
		trackedV2PairsMutex.Unlock()
		return true
	}

	// we have already alerted before
	// therefore we are calculating percentage change

	if pairData.LastAlerted.Add(uniswapV2Options.v2VolumeAlertTimeout).After(now) {
		// don't alert if already alerted (alert timeout)
		// last alerted + timeout > now
		return false
	}

	// make sure previous token volume isnt 0, otherwise percentage will be infinite/undefined
	if tokenPrevVolumeUSD.Cmp(big.NewFloat(0)) == 0 {
		return false
	}

	// volumePercentChange = (tokenVolume - tokenPrevVolume) * 100 / tokenPrevVolume
	volumeChange := big.NewFloat(0).Sub(tokenVolumeUSD, tokenPrevVolumeUSD)
	volumePercentChange := big.NewFloat(0).Quo(big.NewFloat(0).Mul(volumeChange, big.NewFloat(100)), tokenPrevVolumeUSD)
	// debug
	fmt.Printf("[v2] token: %v\n", token.Info.TokenName)
	fmt.Printf("tokenVolume: %v\n", tokenVolume)
	fmt.Printf("tokenPrevVolume: %v\n", tokenPrevVolume)
	fmt.Printf("volumePercentChange: %v\n", volumePercentChange)
	if volumePercentChange.Cmp(uniswapV2Options.v2VolumePercentageThreshold) < 0 {
		// did not exceed threshold, don't alert
		return false
	}

	// update last alerted time
	pairData.LastAlerted = now

	// send discord alert
	v2DiscordWebhook.SendMessage(fmt.Sprintf(
		"## Uniswap V2 Pair - Volume alert\n"+
			"Pair: %v %v\n"+
			"Token0 %v (%v): %v %v\n"+
			"Token1 %v (%v): %v %v\n"+
			"Volume of %v (%v) in last %v: %v USD (prev %v USD)\n"+
			"Volume percentage change: %v%%",
		utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pair), pair.Hex()),
		formatUniswapHyperlink("pair", "uniswap", pair.Hex()),
		utils.CodeQuote(token0.Info.TokenName), utils.CodeQuote(token0.Info.TokenSymbol),
		utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pairData.Token0), pairData.Token0.Hex()),
		formatUniswapHyperlink("token", "uniswap", pairData.Token0.Hex()),
		utils.CodeQuote(token1.Info.TokenName), utils.CodeQuote(token1.Info.TokenSymbol),
		utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pairData.Token1), pairData.Token1.Hex()),
		formatUniswapHyperlink("token", "uniswap", pairData.Token1.Hex()),
		utils.CodeQuote(token.Info.TokenName), utils.CodeQuote(token.Info.TokenSymbol), uniswapV2Options.v2VolumeDuration,
		utils.CodeQuote(tokenVolumeUSD.Text('f', 2)),
		utils.CodeQuote(tokenPrevVolumeUSD.Text('f', 2)),
		utils.CodeQuote(volumePercentChange.Text('f', 2)),
	))

	// update last alerted time
	pairData.LastAlerted = now
	// assign to map
	trackedV2PairsMutex.Lock()
	trackedV2Pairs[pair] = pairData
	trackedV2PairsMutex.Unlock()

	return false
}

// saved tracked pairs data to datafile
func uniswapV2UpdateDataFile() {
	trackedV2PairsMutex.RLock()
	raw, err := json.Marshal(trackedV2Pairs)
	trackedV2PairsMutex.RUnlock()
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(uniswapV2Options.uniswapV2DataFilepath, raw, 0644)
	if err != nil {
		panic(err)
	}
}
