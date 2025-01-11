package actions

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"

	"github.com/Zellic/EVM-trackooor/shared"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
)

var privateKeyHex string
var myNonce uint64
var gasPriceSuggested *big.Int

func (p action) InitSendTransactionTest() {

	privateKeyHex = p.o.CustomOptions["private-key"].(string)

	// get private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		panic(err)
	}

	// get public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	updateNonce(fromAddress)

	// get suggested gas price
	gasPriceSuggested, err = shared.Client.SuggestGasPrice(context.Background())
	if err != nil {
		panic(err)
	}

	for _, address := range p.o.Addresses {
		addTxAddressAction(address, TryBackrunTx)
	}

	// fix stuck txs by sending 0 eth to self
	fmt.Printf("fix stuck txs - sending 0 eth to self\n")
	sendEtherToSelf(privateKeyHex, big.NewInt(0), 2_000_000, gasPriceSuggested)

	myNonce += 1

	// TEMPORARY TESTING

	// (uncomment below to test just sending a tx immediantly after querying block)
	// gasPrice, err := shared.Client.SuggestGasPrice(context.Background())
	// if err != nil {
	// 	panic(err)
	// }

	// pendingBlock, err := shared.GetL2BlockByString("pending")
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("pending block num: %v\n", pendingBlock.Number())
	// sendEtherToSelf(privateKeyHex, big.NewInt(1337), 2_000_000, gasPrice)
}

func updateNonce(addr common.Address) {
	nonce, err := shared.Client.PendingNonceAt(context.Background(), addr)
	if err != nil {
		log.Fatal(err)
	}
	myNonce = nonce
}

func sendEtherToSelf(privateKeyHex string, weiAmount *big.Int, gasLimit uint64, gasPrice *big.Int) {
	// get private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		panic(err)
	}

	// get public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	value := weiAmount // in wei (1 eth)

	nonce := myNonce

	// send eth to self
	toAddress := fromAddress
	var data []byte
	tx := types.NewTransaction(nonce, toAddress, value, gasLimit, gasPrice, data)

	chainID := shared.ChainID

	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), privateKey)
	if err != nil {
		log.Fatal(err)
	}

	// fmt.Printf("sending ether to self\n")
	// fmt.Printf("value: %v\n", value)
	// fmt.Printf("gasLimit: %v\n", gasLimit)
	// fmt.Printf("gasPrice: %v\n", gasPrice)
	err = shared.Client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
}

func TryBackrunTx(p ActionTxData) {
	fmt.Printf("trying to backrun Tx: %v\n", p.Transaction.Hash())
	fmt.Printf("Block: %v\n", p.Block.Number())
	fmt.Printf("from: %v\n", p.From)
	fmt.Printf("to: %v\n", p.To)
	fmt.Printf("Sending ether to self...\n")

	// sendEtherToSelf(
	// 	privateKeyHex,
	// 	big.NewInt(0),
	// 	20052347,
	// 	gasPrice,
	// )

	maxPriorityFee := big.NewInt(1000000)

	newSendEthToSelf(
		privateKeyHex,
		big.NewInt(31),
		big.NewInt(0).Add(gasPriceSuggested, maxPriorityFee),
		gasPriceSuggested,
		maxPriorityFee,
	)

}

func newSendEthToSelf(privateKeyHex string, weiAmount *big.Int, gasLimit *big.Int, gasPrice *big.Int, maxPriorityFee *big.Int) {
	// get private key
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		panic(err)
	}

	// get public key
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		log.Fatal("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	value := weiAmount // in wei

	// send eth to self
	toAddress := fromAddress
	txData := &types.DynamicFeeTx{
		ChainID:    shared.ChainID,
		Nonce:      myNonce,
		GasTipCap:  maxPriorityFee,
		GasFeeCap:  gasLimit,
		Gas:        21000,
		To:         &toAddress,
		Value:      value,
		Data:       nil,
		AccessList: nil,
	}
	signedTx, err := types.SignNewTx(
		privateKey,
		types.LatestSignerForChainID(shared.ChainID),
		txData,
	)
	if err != nil {
		panic(err)
	}

	// fmt.Printf("sending ether to self\n")
	// fmt.Printf("value: %v\n", value)
	// fmt.Printf("gasLimit: %v\n", gasLimit)
	// fmt.Printf("gasPrice: %v\n", gasPrice)
	// fmt.Printf("maxPriorityFee: %v\n", maxPriorityFee)
	err = shared.Client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("tx sent: %s\n", signedTx.Hash().Hex())
	// updateNonce(fromAddress)
	myNonce += 1
}
