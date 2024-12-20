package actions

import (
	"context"
	"encoding/json"
	"evm-trackooor/shared"
	"evm-trackooor/utils"
	"fmt"
	"math/big"
	"os"
	"sync"

	"github.com/ethereum/go-ethereum/common"
)

type tornadoInteractions struct {
	Name                string
	DepositCallers      []common.Address
	WithdrawToAddresses []common.Address
}

type tornadoContractsMutex struct {
	contracts map[common.Address]tornadoInteractions
	mutex     sync.Mutex
}

var tornadoContractsHistorical tornadoContractsMutex

func (actionInfo) InfoTornadoCash() actionInfo {
	name := "TornadoCash"
	overview := "Records historical Tornado.Cash deposits and withdrawals, using emitted events."
	description := "Recorded data include tx sender of Deposit event emitted transactions, " +
		"and the withdrawTo field of withdrawal events, which is the address funds were sent to. \n\n" +
		"Addresses provided should be Tornado.Cash contract addresses."

	options := `"output-filepath" - where to output the JSON data to, default ./tornadoCashAddresses.json`

	example := `"TornadoCash": {
    "addresses": {
        "0x12D66f87A04A9E220743712cE6d9bB1B5616B8Fc": {"name": "Tornado.Cash 0.1 ETH"}
    },
    "options":{
        "output-filepath":"./tornadoCashAddresses.json"
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

func (t *tornadoContractsMutex) addDepositCaller(contract common.Address, caller common.Address) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tornadoInteraction := t.contracts[contract]
	tornadoInteraction.DepositCallers = append(tornadoInteraction.DepositCallers, caller)
	t.contracts[contract] = tornadoInteraction
}

func (t *tornadoContractsMutex) addWithdrawToAddress(contract common.Address, withdrawTo common.Address) {
	t.mutex.Lock()
	defer t.mutex.Unlock()

	tornadoInteraction := t.contracts[contract]
	tornadoInteraction.WithdrawToAddresses = append(tornadoInteraction.WithdrawToAddresses, withdrawTo)
	t.contracts[contract] = tornadoInteraction
}

func InitTornadoContracts() tornadoContractsMutex {
	return tornadoContractsMutex{contracts: make(map[common.Address]tornadoInteractions)}
}

// init function called just before starting trackooor
func (action) InitTornadoCash() {
	tornadoContractsHistorical = InitTornadoContracts()
	addEventSigAction("Withdrawal(address,bytes32,address,uint256)", ProcessTornadoWithdraw)
	addEventSigAction("Deposit(bytes32,uint32,uint256)", ProcessTornadoDeposit)
}

// function called after historical processor is finished
func (p action) FinishedTornadoCash() {
	// wait for async funcs to finish
	RpcWaitGroup.Wait()

	var writeData []byte

	// get tornado contract names from config
	for tornadoAddress := range tornadoContractsHistorical.contracts {
		name := shared.Options.AddressProperties[tornadoAddress]["name"].(string)

		tornadoInteractions := tornadoContractsHistorical.contracts[tornadoAddress]
		tornadoInteractions.Name = name
		tornadoContractsHistorical.contracts[tornadoAddress] = tornadoInteractions
	}

	withdrawToAddressesJSON, _ := json.Marshal(
		struct {
			FromBlock        *big.Int
			ToBlock          *big.Int
			TornadoContracts map[common.Address]tornadoInteractions
		}{
			FromBlock:        shared.Options.HistoricalOptions.FromBlock,
			ToBlock:          shared.Options.HistoricalOptions.ToBlock,
			TornadoContracts: tornadoContractsHistorical.contracts,
		},
	)
	writeData = withdrawToAddressesJSON

	var outputFilepath string
	if v, ok := p.o.CustomOptions["output-filepath"]; ok {
		outputFilepath = v.(string)
	} else {
		outputFilepath = "./tornadoCashAddresses.json"
		fmt.Printf("TornadoCash: Output filepath not specified, using default of %v\n", outputFilepath)
	}

	err := os.WriteFile(outputFilepath, writeData, 0644)
	if err != nil {
		panic(err)
	}
}

func ProcessTornadoWithdraw(p ActionEventData) {
	// validate decoded event have fields we're expecting
	if _, ok := p.DecodedData["to"]; !ok {
		return
	}

	// record address funds were withdrawn to
	contractAddress := p.EventLog.Address
	to := p.DecodedData["to"].(common.Address)

	tornadoContractsHistorical.addWithdrawToAddress(contractAddress, to)
}

func ProcessTornadoDeposit(p ActionEventData) {
	// record address of depositer
	RpcWaitGroup.Add(1)
	go func() {
		contractAddress := p.EventLog.Address

		// query for tx sender
		txHash := p.EventLog.TxHash
		tx, _, err := shared.Client.TransactionByHash(context.Background(), txHash)
		if err != nil {
			panic(err)
		}
		sender := utils.GetTxSender(tx)

		// record tx sender
		tornadoContractsHistorical.addDepositCaller(contractAddress, *sender)

		RpcWaitGroup.Done()
	}()
}
