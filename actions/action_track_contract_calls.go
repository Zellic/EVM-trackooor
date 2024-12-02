package actions

import (
	"bytes"
	"evm-trackooor/shared"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// map func sigs to their corresponding hashes
var trackedFuncSigs map[string][]byte

func (p action) InitTrackContractCalls() {
	trackedFuncSigs = make(map[string][]byte)

	for _, addr := range shared.Options.FilterAddresses {
		addTxAddressAction(addr, checkFunctionCall)
	}

	funcSigsInterface := p.o.CustomOptions["function-signatures"].([]interface{})
	for _, funcSigInterface := range funcSigsInterface {
		funcSig := funcSigInterface.(string)
		funcSigHash := crypto.Keccak256([]byte(funcSig))[:4]
		trackedFuncSigs[funcSig] = funcSigHash
	}
}

func checkFunctionCall(p ActionTxData) {
	// continue only if tx was contract interaction
	txType := shared.DetermineTxType(p.Transaction, p.Block.Number())
	if txType != shared.ContractTx {
		return
	}

	// get tx To and From
	to := p.To
	from := p.From

	// get tx calldata
	txData := p.Transaction.Data()
	// get tx func selector
	txFuncSelector := txData[:4]

	// log if matches tracked func sig
	for trackedFuncSig, trackedFuncSigHash := range trackedFuncSigs {
		if bytes.Equal(trackedFuncSigHash, txFuncSelector) {
			fmt.Printf(
				"%v called function %v on contract %v\n",
				from,
				trackedFuncSig,
				to,
			)
		}
	}
}
