package actions

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/Zellic/EVM-trackooor/shared"

	"github.com/ethereum/go-ethereum/common"
)

var targetBytecodes [][]byte
var rateLimitMutex sync.RWMutex

func (p action) InitTesting() {

	if v, ok := p.o.CustomOptions["bytecodes"]; ok {
		hexBytecodeInterfaces := v.([]interface{})
		for _, hexBytecodeInterface := range hexBytecodeInterfaces {
			hexBytecode := hexBytecodeInterface.(string)
			bytecode := common.Hex2Bytes(hexBytecode)
			targetBytecodes = append(targetBytecodes, bytecode)
		}
	}

	fmt.Printf("Looking for bytecodes:\n")
	for _, bytecode := range targetBytecodes {
		fmt.Printf("%v\n", common.Bytes2Hex(bytecode))
	}

	addBlockAction(blockMined)
}

func blockMined(p ActionBlockData) {
	block := p.Block
	for _, tx := range block.Transactions() {
		// To() returns nil if tx is deployment tx
		if tx.To() == nil {
			// to := tx.To()
			// from := utils.GetTxSender(tx)

			deployedContract, _ := shared.GetDeployedContractAddress(tx)
			contractCode, err := shared.Client.CodeAt(context.Background(), deployedContract, block.Number())
			if err != nil {
				panic(err)
			}

			// fmt.Printf("contract deployed\n")

			for _, bytecode := range targetBytecodes {
				if bytes.Contains(contractCode, bytecode) {
					fmt.Printf("Contract %v contains bytecode %v\n", deployedContract, bytecode)

					err = os.WriteFile("./output_test.out", []byte(fmt.Sprintf(
						"Contract %v contains bytecode %v\n", deployedContract, bytecode,
					)), 0644)
					if err != nil {
						panic(err)
					}
				}
			}

		}
	}
}
