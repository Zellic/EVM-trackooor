package actions

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/Zellic/EVM-trackooor/shared"
	"github.com/Zellic/EVM-trackooor/utils"

	"github.com/Zellic/EVM-trackooor/contracts/uniswapPool"

	"github.com/ethereum/go-ethereum/common"
)

// token structs

type NativePrice struct {
	Value    string `json:"value"`
	Decimals int    `json:"decimals"`
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Address  string `json:"address"`
}

type TokenInfo struct {
	TokenName               string      `json:"tokenName"`
	TokenSymbol             string      `json:"tokenSymbol"`
	TokenLogo               string      `json:"tokenLogo"`
	TokenDecimals           string      `json:"tokenDecimals"`
	NativePrice             NativePrice `json:"nativePrice"`
	UsdPrice                float64     `json:"usdPrice"`
	UsdPriceFormatted       string      `json:"usdPriceFormatted"`
	ExchangeName            string      `json:"exchangeName"`
	ExchangeAddress         string      `json:"exchangeAddress"`
	TokenAddress            string      `json:"tokenAddress"`
	PriceLastChangedAtBlock string      `json:"priceLastChangedAtBlock"`
	PossibleSpam            bool        `json:"possibleSpam"`
	VerifiedContract        bool        `json:"verifiedContract"`
	PairAddress             string      `json:"pairAddress"`
	PairTotalLiquidityUsd   string      `json:"pairTotalLiquidityUsd"`
	PercentChange24hr       string      `json:"24hrPercentChange"`
}

type tokenData struct {
	Info            TokenInfo `json:"token-info"`
	InfoLastUpdated time.Time `json:"token-info-last-updated"` // last time token info was set
}

// tokens and addresses
var USDT common.Address
var USDC common.Address
var WETH common.Address

var wethUsdcV3Pool common.Address

// v3 pool structs

type v3Swap struct {
	// event data
	Sender       common.Address `json:"sender"`
	Recipient    common.Address `json:"recipient"`
	Amount0      *big.Int       `json:"amount0"`
	Amount1      *big.Int       `json:"amount1"`
	SqrtPriceX96 *big.Int       `json:"sqrt-price-x96"`
	Liquidity    *big.Int       `json:"liquidity"`
	Tick         *big.Int       `json:"tick"`
	// other data
	Timestamp time.Time `json:"timestamp"`
}

type v3PoolInfo struct {
	Token0                 common.Address `json:"token0"`                    // token0 address
	Token1                 common.Address `json:"token1"`                    // token1 address
	Token0Volume           *big.Int       `json:"token0-volume"`             // volume of token0
	Token1Volume           *big.Int       `json:"token1-volume"`             // volume of token1
	Token0PrevPeriodVolume *big.Int       `json:"token0-prev-period-volume"` // volume of token0 in the prev X hour period
	Token1PrevPeriodVolume *big.Int       `json:"token1-prev-period-volume"` // volume of token1 in the prev X hour period

	LastAlerted time.Time `json:"last-alerted"` // last time webhook msg was sent

	Swaps []v3Swap `json:"swaps"` // swaps in the last <time duration>
}

// logging
var uniswapV3Log *log.Logger

// discord webhook
var v3DiscordWebhook shared.WebhookInstance

// custom options
var uniswapV3Options struct {
	v3PriceStaleDuration        time.Duration // max time until fetch new token price
	v3VolumeDuration            time.Duration // track volume in last <time duration>
	v3VolumeThresholdUSD        *big.Float
	v3VolumePercentageThreshold *big.Float // alert if volume percentage change from last X hours > threshold
	v3VolumeAlertTimeout        time.Duration
	v3WebhookUrl                string
	uniswapV3DataFilepath       string
}

// shared custom options
// note: this is shared across uniswap V2 volume action
var moralisApiKey string
var moralisChainString string

// map pool address to its info
var trackedV3Pools map[common.Address]v3PoolInfo
var trackedV3PoolsMutex sync.RWMutex

// map token address to its info
// note: this is shared across uniswap V2 volume action
var tokens map[common.Address]tokenData
var tokensMutex sync.RWMutex

func (p action) InitUniswapV3Volume() {
	// init maps
	trackedV3Pools = make(map[common.Address]v3PoolInfo)
	tokens = make(map[common.Address]tokenData)

	// init logging
	uniswapV3Log = shared.TimeLogger("[Uniswap V3] ")

	// track uniswapv3 factory(s)
	for _, address := range p.o.Addresses {
		addAddressEventSigAction(
			address,
			"PoolCreated(address,address,uint24,int24,address)",
			handleV3PoolCreated,
		)
	}

	// retrieve custom options
	uniswapV3GetCustomOptions(p.o.CustomOptions)
}

func uniswapV3GetCustomOptions(CustomOptions map[string]interface{}) {
	// get data filepath
	if v, ok := CustomOptions["data-filepath"]; ok {
		uniswapV3Options.uniswapV3DataFilepath = v.(string)
	} else {
		uniswapV3Log.Fatal("\"data-filepath\" not specified!")
	}
	uniswapV3Log.Printf("using data file %v\n", uniswapV3Options.uniswapV3DataFilepath)

	// get api key
	if v, ok := CustomOptions["moralis-api-key"]; ok {
		moralisApiKey = v.(string)
	} else {
		uniswapV3Log.Fatalf(`Please provide API KEY "moralis-api-key"`)
	}

	if v, ok := CustomOptions["moralis-chain-string"]; ok {
		moralisChainString = v.(string)
	} else {
		uniswapV3Log.Fatalf(`Please provide chain "moralis-chain-string" (e.g. "eth")`)
	}

	if v, ok := CustomOptions["volume-threshold-usd"]; ok {
		s := v.(string)
		uniswapV3Options.v3VolumeThresholdUSD, ok = big.NewFloat(0).SetString(s)
		if !ok {
			uniswapV3Log.Fatalf("Unable to set volumeThresholdUSD with value: %v\n", s)
		}
	} else {
		uniswapV3Log.Fatalf("\"volume-threshold-usd\" not specified!")
	}
	uniswapV3Log.Printf("Using volume threshold USD %v\n", uniswapV3Options.v3VolumeThresholdUSD.Text('f', 0))

	// get discord webhook url
	if v, ok := CustomOptions["webhook-url"]; ok {
		uniswapV3Options.v3WebhookUrl = v.(string)
		v3DiscordWebhook = shared.WebhookInstance{
			WebhookURL:           uniswapV3Options.v3WebhookUrl,
			Username:             "evm trackooor",
			Avatar_url:           "",
			RetrySendingMessages: true,
		}
	} else {
		uniswapV3Log.Fatalf("\"webhook-url\" not specified!")
	}
	uniswapV3Log.Printf("Using discord webhook URL %v\n", uniswapV3Options.v3WebhookUrl)

	// get volume duration
	if v, ok := CustomOptions["volume-duration"]; ok {
		s := v.(string)
		duration, err := time.ParseDuration(s)
		if err != nil {
			uniswapV3Log.Fatalf("Invalid time duration %v\n", s)
		}
		uniswapV3Options.v3VolumeDuration = duration
		uniswapV3Log.Printf("Using volume duration of %v\n", uniswapV3Options.v3VolumeDuration.String())
	} else {
		uniswapV3Options.v3VolumeDuration = time.Hour * 24
		uniswapV3Log.Printf("\"volume-duration\" not set, defaulting to %v\n", uniswapV3Options.v3VolumeDuration)
	}

	// get volume alert timeout
	if v, ok := CustomOptions["volume-alert-timeout"]; ok {
		s := v.(string)
		duration, err := time.ParseDuration(s)
		if err != nil {
			uniswapV3Log.Fatalf("Invalid time duration %v\n", s)
		}
		uniswapV3Options.v3VolumeAlertTimeout = duration
		uniswapV3Log.Printf("Using volume alert timeout of %v\n", uniswapV3Options.v3VolumeAlertTimeout.String())
	} else {
		uniswapV3Options.v3VolumeAlertTimeout = time.Hour * 1
		uniswapV3Log.Printf("\"volume-alert-timeout\" not set, defaulting to %v\n", uniswapV3Options.v3VolumeAlertTimeout)
	}

	// get price stale duration
	if v, ok := CustomOptions["price-stale-duration"]; ok {
		s := v.(string)
		duration, err := time.ParseDuration(s)
		if err != nil {
			uniswapV3Log.Fatalf("Invalid time duration %v\n", s)
		}
		uniswapV3Options.v3PriceStaleDuration = duration
		uniswapV3Log.Printf("Using price stale duration of %v\n", uniswapV3Options.v3PriceStaleDuration.String())
	} else {
		uniswapV3Options.v3PriceStaleDuration = time.Minute * 10
		uniswapV3Log.Printf("\"price-stale-duration\" not set, defaulting to %v\n", uniswapV3Options.v3PriceStaleDuration)
	}

	// get volume percentage change threshold
	if v, ok := CustomOptions["volume-percentage-threshold"]; ok {
		s := v.(string)
		uniswapV3Options.v3VolumePercentageThreshold, ok = big.NewFloat(0).SetString(s)
		if !ok {
			uniswapV3Log.Fatalf("Unable to set volumeThresholdUSD with value: %v\n", s)
		}
		uniswapV3Log.Printf("Using percentage change threshold of %v\n", uniswapV3Options.v3VolumePercentageThreshold.String())
	} else {
		uniswapV3Options.v3VolumePercentageThreshold = big.NewFloat(200)
		uniswapV3Log.Printf("\"volume-percentage-threshold\" not set, defaulting to %v\n", uniswapV3Options.v3VolumePercentageThreshold)
	}

	// set token addresses
	// USDT
	if v, ok := CustomOptions["usdt-address"]; ok {
		USDT = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		USDT = common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	} else {
		uniswapV3Log.Fatalf("\"usdt-address\" not provided! Please provide USDT token address.")
	}

	// USDC
	if v, ok := CustomOptions["usdc-address"]; ok {
		USDC = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		USDC = common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48")
	} else {
		uniswapV3Log.Fatalf("\"usdc-address\" not provided! Please provide USDC token address.")
	}
	// WETH
	if v, ok := CustomOptions["weth-address"]; ok {
		WETH = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		WETH = common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2")
	} else {
		uniswapV3Log.Fatalf("\"weth-address\" not provided! Please provide WETH token address.")
	}

	// get WETH/USDC uniswap v3 pool address
	if v, ok := CustomOptions["weth-usdc-pool"]; ok {
		wethUsdcV3Pool = common.HexToAddress(v.(string))
	} else if shared.ChainID.Cmp(big.NewInt(1)) == 0 {
		// set to address as on ETH chain
		wethUsdcV3Pool = common.HexToAddress("0x88e6A0c2dDD26FEEb64F039a2c41296FcB3f5640")
	} else {
		uniswapV3Log.Fatalf("\"weth-usdc-pool\" not provided! Please provide WETH/USDc uniswap V3 pool address.")
	}

	// load previously saved data file (if exists)
	if _, err := os.Stat(uniswapV3Options.uniswapV3DataFilepath); err == nil {
		// data file exists, load previous data
		dat, err := os.ReadFile(uniswapV3Options.uniswapV3DataFilepath)
		if err != nil {
			panic(err)
		}

		// load JSON data
		trackedV3PoolsMutex.Lock()
		err = json.Unmarshal(dat, &trackedV3Pools)
		if err != nil {
			panic(err)
		}

		uniswapV3Log.Printf("Loaded saved data, tracking %v pools\n", len(trackedV3Pools))
		// add previous tracked pools to tracking
		for pool := range trackedV3Pools {
			addAddressEventSigAction(pool, "Swap(address,address,int256,int256,uint160,uint128,int24)", handleV3PoolSwapEvent)
		}
		trackedV3PoolsMutex.Unlock()

	} else if errors.Is(err, os.ErrNotExist) {
		// data file does not exist, all good
	} else {
		panic(err)
	}
}

func handleV3PoolCreated(p ActionEventData) {
	token0 := p.DecodedTopics["token0"].(common.Address)
	token1 := p.DecodedTopics["token1"].(common.Address)
	// fee := p.DecodedTopics["fee"].(*big.Int)
	// tickSpacing := p.DecodedData["tickSpacing"].(*big.Int)
	pool := p.DecodedData["pool"].(common.Address)

	// fmt.Printf("\nNew Uniswap V3 pool created\n")
	// fmt.Printf("token0: %v\n", token0)
	// fmt.Printf("token1: %v\n", token1)
	// fmt.Printf("fee: %v\n", fee)
	// fmt.Printf("tickSpacing: %v\n", tickSpacing)
	// fmt.Printf("pool: %v\n", pool)

	// track the pool for swaps
	addAddressEventSigAction(pool, "Swap(address,address,int256,int256,uint160,uint128,int24)", handleV3PoolSwapEvent)

	// add pool to map
	trackedV3PoolsMutex.Lock()
	trackedV3Pools[pool] = v3PoolInfo{
		Token0: token0,
		Token1: token1,
	}
	trackedV3PoolsMutex.Unlock()

	// update datafile
	uniswapV3UpdateDataFile()
}

func handleV3PoolSwapEvent(p ActionEventData) {
	pool := p.EventLog.Address

	swapData := v3Swap{
		Sender:       p.DecodedTopics["sender"].(common.Address),
		Recipient:    p.DecodedTopics["recipient"].(common.Address),
		Amount0:      p.DecodedData["amount0"].(*big.Int),
		Amount1:      p.DecodedData["amount1"].(*big.Int),
		SqrtPriceX96: p.DecodedData["sqrtPriceX96"].(*big.Int),
		Liquidity:    p.DecodedData["liquidity"].(*big.Int),
		Tick:         p.DecodedData["tick"].(*big.Int),
		Timestamp:    time.Now(),
	}

	// fmt.Printf("\nSwap event detected!! pool: %v\n", pool)

	// append swap data
	trackedV3PoolsMutex.Lock()
	poolData := trackedV3Pools[pool]
	poolData.Swaps = append(poolData.Swaps, swapData)
	trackedV3Pools[pool] = poolData
	trackedV3PoolsMutex.Unlock()

	// update pool's volume
	updateV3Volume(pool)

	// check pool volume and alert if needed
	checkV3Volume(pool)

	// update datafile
	uniswapV3UpdateDataFile()
}

func updateV3Volume(pool common.Address) {
	// loop through all swaps
	trackedV3PoolsMutex.RLock()
	poolData := trackedV3Pools[pool]
	trackedV3PoolsMutex.RUnlock()

	swaps := poolData.Swaps
	token0Volume := big.NewInt(0)
	token1Volume := big.NewInt(0)
	token0PrevVolume := big.NewInt(0)
	token1PrevVolume := big.NewInt(0)

	now := time.Now()
	removing := []int{} // swap indexes to remove due to being old
	for ind, swap := range swaps {
		// check if swap within timeframe
		swapTime := swap.Timestamp
		// swapTime + volumeDuration > now
		token0AbsAmount := big.NewInt(0).Abs(swap.Amount0)
		token1AbsAmount := big.NewInt(0).Abs(swap.Amount1)
		if swapTime.Add(uniswapV3Options.v3VolumeDuration).After(now) {
			// get amount of token0 & token1 traded and add to volume
			token0Volume.Add(token0Volume, token0AbsAmount)
			token1Volume.Add(token1Volume, token1AbsAmount)
		} else if swapTime.Add(uniswapV3Options.v3VolumeDuration * 2).After(now) {
			// time within previous period (swapTime + volumeDuration*2 > now)
			token0PrevVolume.Add(token0PrevVolume, token0AbsAmount)
			token1PrevVolume.Add(token1PrevVolume, token1AbsAmount)
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
	poolData.Token0Volume = token0Volume
	poolData.Token1Volume = token1Volume
	poolData.Token0PrevPeriodVolume = token0PrevVolume
	poolData.Token1PrevPeriodVolume = token1PrevVolume
	poolData.Swaps = swaps

	// fmt.Printf("token0Volume: %v\n", token0Volume)
	// fmt.Printf("token1Volume: %v\n", token1Volume)

	// assign to map
	trackedV3PoolsMutex.Lock()
	trackedV3Pools[pool] = poolData
	trackedV3PoolsMutex.Unlock()
}

func checkV3Volume(pool common.Address) {
	trackedV3PoolsMutex.RLock()
	poolData := trackedV3Pools[pool]
	trackedV3PoolsMutex.RUnlock()

	// if either token is a common token, we will use that token for price and end there
	// check token0
	if slices.Contains([]common.Address{USDT, USDC, WETH}, poolData.Token0) {
		updateTokenInfo(poolData.Token0)
		token0Alerted := alertV3Volume(pool, poolData.Token0, poolData.Token0Volume, poolData.Token0PrevPeriodVolume)
		if token0Alerted {
			uniswapV3Log.Printf("V3 Pool %v alerted for token0 (shortcut price)\n", pool)
		}
		return
	}
	// check token1
	if slices.Contains([]common.Address{USDT, USDC, WETH}, poolData.Token1) {
		updateTokenInfo(poolData.Token1)
		token1Alerted := alertV3Volume(pool, poolData.Token1, poolData.Token1Volume, poolData.Token1PrevPeriodVolume)
		if token1Alerted {
			uniswapV3Log.Printf("V3 Pool %v alerted for token1 (shortcut price)\n", pool)
		}
		return
	}

	// otherwise we will just fetch prices as usual

	// update token prices
	updateTokenInfo(poolData.Token0)
	updateTokenInfo(poolData.Token1)

	// send alerts to webhook (if volume > threshold)
	// token0
	token0Alerted := alertV3Volume(pool, poolData.Token0, poolData.Token0Volume, poolData.Token0PrevPeriodVolume)
	if token0Alerted {
		uniswapV3Log.Printf("V3 Pool %v alerted for token0\n", pool)
		return
	}
	// token1
	token1Alerted := alertV3Volume(pool, poolData.Token1, poolData.Token1Volume, poolData.Token1PrevPeriodVolume)
	if token1Alerted {
		uniswapV3Log.Printf("V3 Pool %v alerted for token1\n", pool)
		return
	}
}

// alert to webhook (if volume > threshold), returns whether or not alert was sent
func alertV3Volume(pool common.Address, tokenAddress common.Address, tokenVolume *big.Int, tokenPrevVolume *big.Int) bool {
	trackedV3PoolsMutex.RLock()
	poolData := trackedV3Pools[pool]
	trackedV3PoolsMutex.RUnlock()

	tokensMutex.RLock()
	token := tokens[tokenAddress]
	token0 := tokens[poolData.Token0]
	token1 := tokens[poolData.Token1]
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
	if poolData.LastAlerted.IsZero() {
		// we have not alerted before, alert if exceed volume threshold
		if tokenVolumeUSD.Cmp(uniswapV3Options.v3VolumeThresholdUSD) < 0 {
			// did not exceed threshold
			return false
		}
		// send webhook msg
		v3DiscordWebhook.SendMessage(fmt.Sprintf(
			"## Uniswap V3 Pool - Volume alert\n"+
				"Pool: %v %v\n"+
				"Token0 %v (%v): %v %v\n"+
				"Token1 %v (%v): %v %v\n"+
				"Volume of %v (%v) in last %v: %v USD",
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pool), pool.Hex()), // first %v
			formatUniswapHyperlink("pool", "uniswap", pool.Hex()),
			utils.CodeQuote(token0.Info.TokenName), utils.CodeQuote(token0.Info.TokenSymbol),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(poolData.Token0), poolData.Token0.Hex()),
			formatUniswapHyperlink("token", "uniswap", poolData.Token0.Hex()),
			utils.CodeQuote(token1.Info.TokenName), utils.CodeQuote(token1.Info.TokenSymbol),
			utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(poolData.Token1), poolData.Token1.Hex()),
			formatUniswapHyperlink("token", "uniswap", poolData.Token1.Hex()),
			utils.CodeQuote(token.Info.TokenName), utils.CodeQuote(token.Info.TokenSymbol), uniswapV3Options.v3VolumeDuration,
			utils.CodeQuote(tokenVolumeUSD.Text('f', 0)),
		))

		// update last alerted time
		poolData.LastAlerted = now
		// assign to map
		trackedV3PoolsMutex.Lock()
		trackedV3Pools[pool] = poolData
		trackedV3PoolsMutex.Unlock()
		return true
	}

	// we have already alerted before
	// therefore we are calculating percentage change

	// don't alert if already alerted (alert timeout)
	// last alerted + timeout > now
	if poolData.LastAlerted.Add(uniswapV3Options.v3VolumeAlertTimeout).After(now) {
		return false
	}

	// make sure previous token volume isnt 0, otherwise percentage will be infinite/undefined
	if tokenPrevVolumeUSD.Cmp(big.NewFloat(0)) == 0 {
		return false
	}

	// calculate percentage change based on USD amount
	// volumePercentChange = (tokenVolume - tokenPrevVolume) * 100 / tokenPrevVolume
	volumeChange := big.NewFloat(0).Sub(tokenVolumeUSD, tokenPrevVolumeUSD)
	volumePercentChange := big.NewFloat(0).Quo(big.NewFloat(0).Mul(volumeChange, big.NewFloat(100)), tokenPrevVolumeUSD)
	// debug
	fmt.Printf("[v3] token: %v\n", token.Info.TokenName)
	fmt.Printf("tokenVolume: %v\n", tokenVolume)
	fmt.Printf("tokenPrevVolume: %v\n", tokenPrevVolume)
	fmt.Printf("volumePercentChange: %v\n", volumePercentChange)
	if volumePercentChange.Cmp(uniswapV3Options.v3VolumePercentageThreshold) < 0 {
		// did not exceed threshold, don't alert
		return false
	}

	// update last alerted time
	poolData.LastAlerted = now

	// send discord alert
	v3DiscordWebhook.SendMessage(fmt.Sprintf(
		"## Uniswap V3 Pool - Volume alert\n"+
			"Pool: %v %v\n"+
			"Token0 %v (%v): %v %v\n"+
			"Token1 %v (%v): %v %v\n"+
			"Volume of %v (%v) in last %v: %v USD (prev %v USD)\n"+
			"Volume percentage change: %v%%",
		utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(pool), pool.Hex()), // first %v
		formatUniswapHyperlink("pool", "uniswap", pool.Hex()),
		utils.CodeQuote(token0.Info.TokenName), utils.CodeQuote(token0.Info.TokenSymbol),
		utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(poolData.Token0), poolData.Token0.Hex()),
		formatUniswapHyperlink("token", "uniswap", poolData.Token0.Hex()),
		utils.CodeQuote(token1.Info.TokenName), utils.CodeQuote(token1.Info.TokenSymbol),
		utils.FormatBlockscanHyperlink("address", utils.ShortenAddress(poolData.Token1), poolData.Token1.Hex()),
		formatUniswapHyperlink("token", "uniswap", poolData.Token1.Hex()),
		utils.CodeQuote(token.Info.TokenName), utils.CodeQuote(token.Info.TokenSymbol), uniswapV3Options.v3VolumeDuration,
		utils.CodeQuote(tokenVolumeUSD.Text('f', 2)),
		utils.CodeQuote(tokenPrevVolumeUSD.Text('f', 2)),
		utils.CodeQuote(volumePercentChange.Text('f', 2)),
	))

	// assign to map
	trackedV3PoolsMutex.Lock()
	trackedV3Pools[pool] = poolData
	trackedV3PoolsMutex.Unlock()

	return false
}

// saved tracked pools data to datafile
func uniswapV3UpdateDataFile() {
	trackedV3PoolsMutex.RLock()
	raw, err := json.Marshal(trackedV3Pools)
	trackedV3PoolsMutex.RUnlock()
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(uniswapV3Options.uniswapV3DataFilepath, raw, 0644)
	if err != nil {
		panic(err)
	}
}

// shared between v2 and v3 volume actions

// return's USD price a given the token and amount of tokens
func calculateUSDpriceOfTokens(tokenAddress common.Address, amountBig *big.Int) *big.Float {
	// update token price
	updateTokenInfo(tokenAddress)

	// get token info
	tokensMutex.RLock()
	token := tokens[tokenAddress]
	tokensMutex.RUnlock()

	// compute token's volume in USD
	amount := big.NewFloat(0).SetInt(amountBig)
	tokenUSDprice := big.NewFloat(0).SetFloat64(token.Info.UsdPrice)
	tokenDecimals, _ := big.NewInt(0).SetString(token.Info.TokenDecimals, 10)
	tokenDecimalsDivisor := big.NewFloat(0).SetInt(big.NewInt(0).Exp(big.NewInt(10), tokenDecimals, nil))
	// convert token volume to USD volume, taking into account token decimals
	// (tokenVolume/tokenDecimals) * tokenUSDprice
	tokenVolumeUSD := big.NewFloat(0).Quo(big.NewFloat(0).Mul(amount, tokenUSDprice), tokenDecimalsDivisor)
	return tokenVolumeUSD
}

// updates token's tokenInfo if stale
func updateTokenInfo(tokenAddress common.Address) {
	// check if token info already exists
	tokensMutex.RLock()
	token := tokens[tokenAddress]
	tokensMutex.RUnlock()

	now := time.Now()
	lastUpdated := token.InfoLastUpdated

	// if price is stale, update
	// lastUpdated + staleDuration < now
	if lastUpdated.Add(uniswapV3Options.v3PriceStaleDuration).Before(now) {
		// fetch token info
		tokenInfo, err := getTokenInfo(tokenAddress)
		if err != nil {
			uniswapV3Log.Printf("Unable to get token info for token %v, err %v\n", tokenAddress, err)
			return
		}
		uniswapV3Log.Printf("Updated token info for %v (%v) USD price: %v\n", tokenInfo.TokenName, tokenInfo.TokenSymbol, tokenInfo.UsdPrice)

		// tokenData
		data := tokenData{
			Info:            tokenInfo,
			InfoLastUpdated: now,
		}

		// assign to map
		tokensMutex.Lock()
		tokens[tokenAddress] = data
		tokensMutex.Unlock()
	}
}

func getTokenInfo(tokenAddress common.Address) (TokenInfo, error) {
	// if its common tokens (WETH, USDC, USDT)
	// query WETH/USDC uniswap pool for price of WETH
	// or return USDC and USDT price as 1
	switch tokenAddress {
	case USDT: // USDT
		return TokenInfo{
			TokenName:   "Tether USD",
			TokenSymbol: "USDT",
			UsdPrice:    1,
		}, nil
	case USDC: // USDC
		return TokenInfo{
			TokenName:   "USDC",
			TokenSymbol: "USDC",
			UsdPrice:    1,
		}, nil
	case WETH: // WETH
		poolAddr := wethUsdcV3Pool // WETH USDC uniswap v3 pool
		wethUsdcPool, err := uniswapPool.NewUniswapV3Pool(poolAddr, shared.Client)
		if err != nil {
			uniswapV3Log.Printf("Error: failed to get WETH price (usdc) from v3 pool")
			break
		}
		slot0, err := wethUsdcPool.Slot0(nil)
		sqrtPriceX96 := slot0.SqrtPriceX96
		if err != nil {
			uniswapV3Log.Printf("Error: failed to get WETH price (usdc) from v3 pool")
			break
		}
		sqrtPriceX96Float := big.NewFloat(0).SetInt(sqrtPriceX96)
		// ethPrice = 1e12 * (sqrtPriceX96Float / 2**96)^2
		twoTo96 := big.NewInt(0).Exp(big.NewInt(2), big.NewInt(96), nil)
		twoTo96Float := big.NewFloat(0).SetInt(twoTo96)
		sqrtPrice := big.NewFloat(0).Quo(sqrtPriceX96Float, twoTo96Float)
		ethPrice := big.NewFloat(0).Quo(big.NewFloat(1e12), big.NewFloat(0).Mul(sqrtPrice, sqrtPrice))
		ethPriceFloat64, _ := ethPrice.Float64()
		return TokenInfo{
			TokenName:   "Wrapped Ether",
			TokenSymbol: "WETH",
			UsdPrice:    ethPriceFloat64,
		}, nil
	}

	// otherwise, fallback to querying moralis api

	// TODO this is only for ETH rn, look into extending it to other chains?
	url := fmt.Sprintf(
		"https://deep-index.moralis.io/api/v2.2/erc20/%v/price?chain=%v&include=percent_change",
		tokenAddress, moralisChainString,
	)

	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return TokenInfo{}, err
	}

	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-API-Key", moralisApiKey)

	resp, err := client.Do(req)
	if err != nil {
		return TokenInfo{}, err
	}
	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		return TokenInfo{}, err
	}

	var tokenInfo TokenInfo
	err = json.Unmarshal(bodyText, &tokenInfo)
	if err != nil {
		return TokenInfo{}, err
	}
	return tokenInfo, nil
}

func formatUniswapHyperlink(typ string, displayValue string, value string) string {
	url := uniswapFormat(typ, value)
	// dont format if url wasnt formatted
	if url == value {
		return displayValue
	}
	return fmt.Sprintf("[`%v`](<%v>)", displayValue, url)
}

func uniswapFormat(typ string, value string) string {
	// TODO this is only for ETH rn, look into extending it to other chains?
	switch typ {
	case "token":
		return fmt.Sprintf("https://app.uniswap.org/explore/tokens/ethereum/%v", value)
	case "pool", "pair":
		return fmt.Sprintf("https://app.uniswap.org/explore/pools/ethereum/%v", value)
	}
	return value
}
