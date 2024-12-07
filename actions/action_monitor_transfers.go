package actions

import (
	"evm-trackooor/shared"
	"evm-trackooor/utils"
	"fmt"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
)

var monitoredAddresses []common.Address

func (p action) InitMonitorTransfers() {
	// monitor addresses for transactions
	monitoredAddresses = p.o.Addresses
	for _, addr := range monitoredAddresses {
		addTxAddressAction(addr, handleAddressTx)
	}
	// monitor erc20 token for transfer events
	for _, addrInterface := range p.o.CustomOptions["erc20-tokens"].([]interface{}) {
		erc20TokenAddress := common.HexToAddress(addrInterface.(string))
		addAddressEventSigAction(erc20TokenAddress, "Transfer(address,address,uint256)", handleTokenTransfer)
	}
}

// called when a tx is from/to monitored address
func handleAddressTx(p ActionTxData) {
	from := *p.From
	value := p.Transaction.Value()
	// tx is from monitored address (since it can be either from/to)
	// and value of tx > 0
	if slices.Contains(monitoredAddresses, from) &&
		value.Cmp(big.NewInt(0)) > 0 {
		// alert
		fmt.Printf("Native ETH Transfer by %v with value %v\n", from, utils.FormatDecimals(value, 18))
	}
}

// called when erc20 token we're tracking emits Transfer event
func handleTokenTransfer(p ActionEventData) {
	from := p.DecodedTopics["from"].(common.Address)
	value := p.DecodedData["value"].(*big.Int)

	// erc20 transfer is from an address we're monitoring
	if slices.Contains(monitoredAddresses, from) {
		// get erc20 token info (to format value decimals + symbol)
		token := p.EventLog.Address
		tokenInfo := shared.RetrieveERC20Info(token)
		decimals := tokenInfo.Decimals
		symbol := tokenInfo.Symbol

		// alert
		fmt.Printf("ERC20 Transfer by %v with value %v %v\n", from, utils.FormatDecimals(value, decimals), symbol)
	}
}
