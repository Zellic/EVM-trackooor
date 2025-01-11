package shared

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"math/big"
	"reflect"
	"sync"

	"github.com/Zellic/EVM-trackooor/contracts/IERC20Metadata"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	DeploymentTx = iota
	RegularTx
	ContractTx
)

var TxTypes = map[int]string{
	DeploymentTx: "Deployment",
	RegularTx:    "EOA",
	ContractTx:   "Contract",
}

// cache blocks as to not query same block number multiple times
// warning: this may use a lot of memory
var CachedBlocks map[*big.Int]*types.Header
var CachedBlocksMutex = sync.RWMutex{} // prevent race condition for map CachedBlocks

func GetBlockHeader(blockNum *big.Int) *types.Header {
	var block *types.Header
	var err error

	CachedBlocksMutex.RLock()
	if v, ok := CachedBlocks[blockNum]; ok {
		CachedBlocksMutex.RUnlock()

		block = v
	} else {
		CachedBlocksMutex.RUnlock()

		block, err = Client.HeaderByNumber(context.Background(), blockNum)
		if err != nil {
			log.Fatalf(`Failed to retrieve info for block %v, error "%v"\n`, blockNum, err)
		}
		CachedBlocksMutex.Lock()
		CachedBlocks[blockNum] = block
		CachedBlocksMutex.Unlock()
	}
	return block
}

// determine action type of transaction.
// requires 1 RPC request unless cache used.
// not to be confused with numerical types such as for legacy tx, access list tx etc.
// three types possible: regular (EOA to EOA), deployment, execution (of smart contracts)
func DetermineTxType(tx *types.Transaction, blockNumber *big.Int) int {
	if tx.To() == nil { // contract deployment
		return DeploymentTx
	}
	to := *tx.To()
	return DetermineAddressType(to, blockNumber)
}

// returns whether an address is a contract or EOA, at a block number, costing 1 query if address uncached.
// result is cached, which however will not take into the block number. assumes contracts stay as contracts
func DetermineAddressType(address common.Address, blockNumber *big.Int) int {
	// check if we already know address type in cache
	// although cache does not take into account block number, we will assume
	// addresses that were contracts will stay as contracts
	AddressTypeCacheMutex.RLock()
	if v, ok := AddressTypeCache[address]; ok {
		AddressTypeCacheMutex.RUnlock()
		return v
	}
	AddressTypeCacheMutex.RUnlock()

	// check if 'to' has bytecode. if not, then it is an EOA
	// caches result in `AddressTypeCache`
	bytecode, err := Client.CodeAt(context.Background(), address, blockNumber)
	if err != nil {
		log.Fatal(err)
	}
	if len(bytecode) == 0 {
		AddressTypeCacheMutex.Lock()
		AddressTypeCache[address] = RegularTx
		AddressTypeCacheMutex.Unlock()
		return RegularTx
	}
	AddressTypeCacheMutex.Lock()
	AddressTypeCache[address] = ContractTx
	AddressTypeCacheMutex.Unlock()
	return ContractTx
}

// returns deployed contract address given a tx
func GetDeployedContractAddress(tx *types.Transaction) (common.Address, error) {
	txReceipt, err := Client.TransactionReceipt(context.Background(), tx.Hash())
	if err != nil {
		// log.Fatalf("Failed to get transaction receipt of tx %v: %v", tx.Hash(), err)
		Warnf(slog.Default(), "Failed to get transaction receipt of tx %v: %v", tx.Hash(), err)
	}
	return txReceipt.ContractAddress, err
}

func BytesToHex(data interface{}) (string, error) {
	v := reflect.ValueOf(data)

	// Ensure the value is of slice or array kind
	if v.Kind() != reflect.Slice && v.Kind() != reflect.Array {
		return "", fmt.Errorf("unsupported type: %s, expected slice or array", v.Kind().String())
	}

	// Ensure the elements are of the type uint8 (aka byte)
	if v.Type().Elem().Kind() != reflect.Uint8 {
		return "", fmt.Errorf("unsupported element type: %s, expected uint8 (byte)", v.Type().Elem().Kind().String())
	}

	// Create a byte slice from the reflected value
	bytes := make([]byte, v.Len())
	for i := 0; i < v.Len(); i++ {
		bytes[i] = byte(v.Index(i).Uint())
	}

	// Convert the byte slice to a hexadecimal string
	return hex.EncodeToString(bytes), nil
}

func RetrieveERC20Info(tokenAddress common.Address) ERC20Info {
	var erc20Info ERC20Info

	// check if we already fetch the info. if so, returned the cached info
	if v, ok := ERC20TokenInfos[tokenAddress]; ok {
		return v
	}

	tokenInstance, err := IERC20Metadata.NewIERC20Metadata(tokenAddress, Client)
	if err != nil {
		log.Fatal(err)
	}

	Infof(slog.Default(), "Retrieving ERC20 token info...\n")
	tokenName, err := tokenInstance.Name(&bind.CallOpts{})
	if err != nil {
		Warnf(slog.Default(), "Could not retrieve token name of %v!\n", tokenAddress)
	}

	tokenDecimals, err := tokenInstance.Decimals(&bind.CallOpts{})
	if err != nil {
		Warnf(slog.Default(), "Could not retrieve token decimals of %v!\n", tokenAddress)
	}

	tokenSymbol, err := tokenInstance.Symbol(&bind.CallOpts{})
	if err != nil {
		Warnf(slog.Default(), "Could not retrieve token symbol of %v!\n", tokenAddress)
	}

	Infof(slog.Default(), "Retrieved token name '%v'\n", tokenName)
	Infof(slog.Default(), "Retrieved token decimals '%v'\n", tokenDecimals)
	Infof(slog.Default(), "Retrieved token symbol '%v'\n", tokenSymbol)

	erc20Info.Name = tokenName
	erc20Info.Symbol = tokenSymbol
	erc20Info.Decimals = tokenDecimals
	erc20Info.Address = tokenAddress

	// cache to ERC20TokenInfos mapping so we don't retrieve again
	ERC20TokenInfos[tokenAddress] = erc20Info

	return erc20Info
}
