package utils

import (
	"context"
	"encoding/json"
	"evm-trackooor/contracts/IERC20Metadata"
	"evm-trackooor/shared"
	"io"
	"log"
	"log/slog"
	"math/big"
	"net/http"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// returns whether a tx is a deployment or not
// useful instead of `determineTxType` as it does not require a query
func IsDeploymentTx(tx *types.Transaction) bool {
	return tx.To() == nil
}

func RemoveDuplicates[T comparable](sliceList []T) []T {
	allKeys := make(map[T]bool)
	list := []T{}
	for _, item := range sliceList {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}

// returns who sent the transaction
func GetTxSender(tx *types.Transaction) *common.Address {
	// TODO get system tx sender properly
	// prevent crashing when tx type is unsupported like for l2 chains
	v, r, s := tx.RawSignatureValues()
	zero := big.NewInt(0)
	if v == nil || r == nil || s == nil {
		shared.Infof(slog.Default(), "system/invalid tx type handled\n")
		return &common.Address{}
	}
	if v.Cmp(zero) == 0 && r.Cmp(zero) == 0 && s.Cmp(zero) == 0 {
		shared.Infof(slog.Default(), "system/invalid tx type handled\n")
		return &common.Address{}
	}

	newSignerFuncs := []func(*big.Int) types.Signer{
		types.NewCancunSigner,
		types.NewLondonSigner,
		types.NewEIP2930Signer,
	}
	for _, newSignerFunc := range newSignerFuncs {
		from, err := types.Sender(newSignerFunc(shared.ChainID), tx)
		if err == nil {
			return &from
		}
	}

	from, err := types.NewEIP155Signer(shared.ChainID).Sender(tx)
	if err != nil {
		log.Fatalf("Could not get sender (from) of transaction: %v\n", err)
	}
	return &from
}

// contract USD price

type AddressBalanceRetriever struct {
	// map token symbol to its price (USD) for 1 token
	tokenRates map[string]float64
	// make token address to its info (decimals, symbol, etc.)
	tokenInfos map[common.Address]shared.ERC20Info
	// www.coinapi.io api key
	coinApiKey string
}

// returns AddressBalanceRetriever
// can be used to get USD balance of an address,
// which is native ETH value + ERC20 token value
// you specify which ERC20 tokens to check balance for
// uses coinAPI to get token price, not all ERC20 tokens will have price data
func NewAddressBalanceRetriever(coinApiKey string, tokens []common.Address) *AddressBalanceRetriever {
	_tokenRates := make(map[string]float64)
	_tokenInfos := make(map[common.Address]shared.ERC20Info)

	abr := &AddressBalanceRetriever{
		tokenRates: _tokenRates,
		tokenInfos: _tokenInfos,
		coinApiKey: coinApiKey,
	}

	// to set tokenRates
	abr.UpdateTokenPrices()

	// to set tokenInfos
	for _, tokenAddress := range tokens {
		erc20Info := shared.RetrieveERC20Info(tokenAddress)

		symbol := erc20Info.Symbol
		symbol = strings.ToUpper(symbol) //coinAPI uses uppercase symbols

		if _, ok := abr.tokenRates[symbol]; !ok {
			log.Printf("AddressBalanceRetriever - Token %v at %v does not have price data!\n", symbol, tokenAddress)
			continue
		}

		abr.tokenInfos[tokenAddress] = erc20Info
	}

	return abr
}

// calculates balance of contract, in USD, balance = native ETH bal + token bal
func (a *AddressBalanceRetriever) GetAddressUSDbalance(contract common.Address, blockNum *big.Int) float64 {
	var totalUSD float64

	// native ETH
	wei, _ := shared.Client.BalanceAt(context.Background(), contract, blockNum)
	divisor := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(18), nil)
	ether, _ := wei.Div(wei, divisor).Float64()
	totalUSD += ether * a.tokenRates["ETH"]
	if totalUSD != 0 {
		// fmt.Printf("totalUSD: %v\n", totalUSD)
		// fmt.Printf("ether: %v\n", ether)
	}

	// tokens
	for tokenAddress, tokenInfo := range a.tokenInfos {
		balance, err := a.GetERC20BalanceOf(tokenAddress, contract)
		if err != nil {
			log.Printf("AddressBalanceRetriever - Error when getting erc20 balance of! %v\n", err)
			continue
		}
		decimals := tokenInfo.Decimals
		symbol := tokenInfo.Symbol
		symbol = strings.ToUpper(symbol)

		divisor := big.NewInt(0).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
		amount, _ := wei.Div(balance, divisor).Float64()
		totalUSD += amount * a.tokenRates[symbol]
		if amount != 0 {
			// fmt.Printf("amount: %v\n", amount)
			// fmt.Printf("symbol: %v\n", symbol)
			// fmt.Printf("balance: %v\n", balance)
			// fmt.Printf("totalUSD: %v\n", totalUSD)
		}
	}

	return totalUSD
}

func (a *AddressBalanceRetriever) GetERC20BalanceOf(token common.Address, address common.Address) (*big.Int, error) {
	tokenInstance, err := IERC20Metadata.NewIERC20Metadata(token, shared.Client)
	if err != nil {
		panic(err)
	}
	balance, err := tokenInstance.BalanceOf(&bind.CallOpts{}, address)
	if err != nil {
		return big.NewInt(0), err
	}
	return balance, nil
}

// token prices

type Rate struct {
	Time         string  `json:"time"`
	AssetIDQuote string  `json:"asset_id_quote"`
	Rate         float64 `json:"rate"`
}

type CurrentRatesData struct {
	AssetIDBase string `json:"asset_id_base"`
	Rates       []Rate `json:"rates"`
}

func (a *AddressBalanceRetriever) UpdateTokenPrices() {

	url := "https://rest.coinapi.io/v1/exchangerate/USD?invert=true"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		log.Printf("AddressBalanceRetriever - err when update token prices: %v\n", err)
		return
	}
	req.Header.Add("Accept", "text/plain")
	req.Header.Add("X-CoinAPI-Key", a.coinApiKey)

	res, err := client.Do(req)
	if err != nil {
		log.Printf("AddressBalanceRetriever - err when update token prices: %v\n", err)
		return
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Printf("AddressBalanceRetriever - err when update token prices: %v\n", err)
		return
	}

	var currentRates CurrentRatesData
	json.Unmarshal(body, &currentRates)

	for _, rate := range currentRates.Rates {
		a.tokenRates[rate.AssetIDQuote] = rate.Rate
	}

	log.Printf("AddressBalanceRetriever - Updated token prices for %v tokens\n", len(a.tokenRates))
}
