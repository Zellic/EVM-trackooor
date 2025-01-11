package actions

import (
	"fmt"
	"math/big"

	"github.com/Zellic/EVM-trackooor/utils"

	"github.com/ethereum/go-ethereum/common"
)

func (actionInfo) InfoTetherTrack() actionInfo {
	name := "TetherTrack"
	overview := "Simple, example action that logs tether transfers of over 100,000 USDT"

	description := "This is done by listening to Transfer events and checking the value field."

	options := `No options`

	example := `"TetherTrack": {
    "addresses": {},
    "options":{}
}`

	return actionInfo{
		ActionName:          name,
		ActionOverview:      overview,
		ActionDescription:   description,
		ActionOptionDetails: options,
		ActionConfigExample: example,
	}
}

func (p action) InitTetherTrack() {
	tether := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	addAddressEventSigAction(tether, "Transfer(address,address,uint256)", handleTetherTransfer)
}

func handleTetherTransfer(p ActionEventData) {
	to := p.DecodedTopics["to"].(common.Address)
	from := p.DecodedTopics["from"].(common.Address)
	value := p.DecodedData["value"].(*big.Int)

	threshold, _ := big.NewInt(0).SetString("100000000000", 10) // 100k USDT

	if value.Cmp(threshold) >= 0 {
		fmt.Printf(
			"Threshold exceeded, from: %v to: %v value: %v txhash: %v\n",
			to, from, utils.FormatDecimals(value, 6), p.EventLog.TxHash,
		)
	}
}
