// Code generated - DO NOT EDIT.
// This file is a generated binding and any manual changes will be lost.

package multicall2

import (
	"errors"
	"math/big"
	"strings"

	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
)

// Reference imports to suppress errors if they are not otherwise used.
var (
	_ = errors.New
	_ = big.NewInt
	_ = strings.NewReader
	_ = ethereum.NotFound
	_ = bind.Bind
	_ = common.Big1
	_ = types.BloomLookup
	_ = event.NewSubscription
	_ = abi.ConvertType
)

// Multicall2Call is an auto generated low-level Go binding around an user-defined struct.
type Multicall2Call struct {
	Target   common.Address
	CallData []byte
}

// Multicall2Result is an auto generated low-level Go binding around an user-defined struct.
type Multicall2Result struct {
	Success    bool
	ReturnData []byte
}

// Multicall2MetaData contains all meta data concerning the Multicall2 contract.
var Multicall2MetaData = &bind.MetaData{
	ABI: "[{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall2.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"aggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes[]\",\"name\":\"returnData\",\"type\":\"bytes[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall2.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"blockAndAggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall2.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"name\":\"getBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getBlockNumber\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockCoinbase\",\"outputs\":[{\"internalType\":\"address\",\"name\":\"coinbase\",\"type\":\"address\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockDifficulty\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"difficulty\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockGasLimit\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"gaslimit\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getCurrentBlockTimestamp\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"timestamp\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"address\",\"name\":\"addr\",\"type\":\"address\"}],\"name\":\"getEthBalance\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"balance\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"getLastBlockHash\",\"outputs\":[{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"requireSuccess\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall2.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"tryAggregate\",\"outputs\":[{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall2.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"bool\",\"name\":\"requireSuccess\",\"type\":\"bool\"},{\"components\":[{\"internalType\":\"address\",\"name\":\"target\",\"type\":\"address\"},{\"internalType\":\"bytes\",\"name\":\"callData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall2.Call[]\",\"name\":\"calls\",\"type\":\"tuple[]\"}],\"name\":\"tryBlockAndAggregate\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"blockNumber\",\"type\":\"uint256\"},{\"internalType\":\"bytes32\",\"name\":\"blockHash\",\"type\":\"bytes32\"},{\"components\":[{\"internalType\":\"bool\",\"name\":\"success\",\"type\":\"bool\"},{\"internalType\":\"bytes\",\"name\":\"returnData\",\"type\":\"bytes\"}],\"internalType\":\"structMulticall2.Result[]\",\"name\":\"returnData\",\"type\":\"tuple[]\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]",
	Bin: "0x608060405234801561001057600080fd5b506110e4806100206000396000f3fe608060405234801561001057600080fd5b50600436106100b45760003560e01c806372425d9d1161007157806372425d9d146101a657806386d516e8146101c4578063a8b0574e146101e2578063bce38bd714610200578063c3077fa914610230578063ee82ac5e14610262576100b4565b80630f28c97d146100b9578063252dba42146100d757806327e86d6e14610108578063399542e91461012657806342cbb15c146101585780634d2301cc14610176575b600080fd5b6100c1610292565b6040516100ce91906106a3565b60405180910390f35b6100f160048036038101906100ec91906109d2565b61029a565b6040516100ff929190610b5c565b60405180910390f35b610110610423565b60405161011d9190610ba5565b60405180910390f35b610140600480360381019061013b9190610bf8565b610438565b60405161014f93929190610d62565b60405180910390f35b610160610457565b60405161016d91906106a3565b60405180910390f35b610190600480360381019061018b9190610da0565b61045f565b60405161019d91906106a3565b60405180910390f35b6101ae610480565b6040516101bb91906106a3565b60405180910390f35b6101cc610488565b6040516101d991906106a3565b60405180910390f35b6101ea610490565b6040516101f79190610ddc565b60405180910390f35b61021a60048036038101906102159190610bf8565b610498565b6040516102279190610df7565b60405180910390f35b61024a600480360381019061024591906109d2565b610640565b60405161025993929190610d62565b60405180910390f35b61027c60048036038101906102779190610e45565b610663565b6040516102899190610ba5565b60405180910390f35b600042905090565b60006060439150825167ffffffffffffffff8111156102bc576102bb6106e8565b5b6040519080825280602002602001820160405280156102ef57816020015b60608152602001906001900390816102da5790505b50905060005b835181101561041d5760008085838151811061031457610313610e72565b5b60200260200101516000015173ffffffffffffffffffffffffffffffffffffffff1686848151811061034957610348610e72565b5b6020026020010151602001516040516103629190610edd565b6000604051808303816000865af19150503d806000811461039f576040519150601f19603f3d011682016040523d82523d6000602084013e6103a4565b606091505b5091509150816103e9576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016103e090610f51565b60405180910390fd5b808484815181106103fd576103fc610e72565b5b60200260200101819052505050808061041590610fa0565b9150506102f5565b50915091565b60006001436104329190610fe8565b40905090565b60008060604392504340915061044e8585610498565b90509250925092565b600043905090565b60008173ffffffffffffffffffffffffffffffffffffffff16319050919050565b600044905090565b600045905090565b600041905090565b6060815167ffffffffffffffff8111156104b5576104b46106e8565b5b6040519080825280602002602001820160405280156104ee57816020015b6104db61066e565b8152602001906001900390816104d35790505b50905060005b82518110156106395760008084838151811061051357610512610e72565b5b60200260200101516000015173ffffffffffffffffffffffffffffffffffffffff1685848151811061054857610547610e72565b5b6020026020010151602001516040516105619190610edd565b6000604051808303816000865af19150503d806000811461059e576040519150601f19603f3d011682016040523d82523d6000602084013e6105a3565b606091505b509150915085156105ef57816105ee576040517f08c379a00000000000000000000000000000000000000000000000000000000081526004016105e59061108e565b60405180910390fd5b5b604051806040016040528083151581526020018281525084848151811061061957610618610e72565b5b60200260200101819052505050808061063190610fa0565b9150506104f4565b5092915050565b6000806060610650600185610438565b8093508194508295505050509193909250565b600081409050919050565b6040518060400160405280600015158152602001606081525090565b6000819050919050565b61069d8161068a565b82525050565b60006020820190506106b86000830184610694565b92915050565b6000604051905090565b600080fd5b600080fd5b600080fd5b6000601f19601f8301169050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052604160045260246000fd5b610720826106d7565b810181811067ffffffffffffffff8211171561073f5761073e6106e8565b5b80604052505050565b60006107526106be565b905061075e8282610717565b919050565b600067ffffffffffffffff82111561077e5761077d6106e8565b5b602082029050602081019050919050565b600080fd5b600080fd5b600080fd5b600073ffffffffffffffffffffffffffffffffffffffff82169050919050565b60006107c98261079e565b9050919050565b6107d9816107be565b81146107e457600080fd5b50565b6000813590506107f6816107d0565b92915050565b600080fd5b600067ffffffffffffffff82111561081c5761081b6106e8565b5b610825826106d7565b9050602081019050919050565b82818337600083830152505050565b600061085461084f84610801565b610748565b9050828152602081018484840111156108705761086f6107fc565b5b61087b848285610832565b509392505050565b600082601f830112610898576108976106d2565b5b81356108a8848260208601610841565b91505092915050565b6000604082840312156108c7576108c6610794565b5b6108d16040610748565b905060006108e1848285016107e7565b600083015250602082013567ffffffffffffffff81111561090557610904610799565b5b61091184828501610883565b60208301525092915050565b600061093061092b84610763565b610748565b905080838252602082019050602084028301858111156109535761095261078f565b5b835b8181101561099a57803567ffffffffffffffff811115610978576109776106d2565b5b80860161098589826108b1565b85526020850194505050602081019050610955565b5050509392505050565b600082601f8301126109b9576109b86106d2565b5b81356109c984826020860161091d565b91505092915050565b6000602082840312156109e8576109e76106c8565b5b600082013567ffffffffffffffff811115610a0657610a056106cd565b5b610a12848285016109a4565b91505092915050565b600081519050919050565b600082825260208201905092915050565b6000819050602082019050919050565b600081519050919050565b600082825260208201905092915050565b60005b83811015610a81578082015181840152602081019050610a66565b60008484015250505050565b6000610a9882610a47565b610aa28185610a52565b9350610ab2818560208601610a63565b610abb816106d7565b840191505092915050565b6000610ad28383610a8d565b905092915050565b6000602082019050919050565b6000610af282610a1b565b610afc8185610a26565b935083602082028501610b0e85610a37565b8060005b85811015610b4a5784840389528151610b2b8582610ac6565b9450610b3683610ada565b925060208a01995050600181019050610b12565b50829750879550505050505092915050565b6000604082019050610b716000830185610694565b8181036020830152610b838184610ae7565b90509392505050565b6000819050919050565b610b9f81610b8c565b82525050565b6000602082019050610bba6000830184610b96565b92915050565b60008115159050919050565b610bd581610bc0565b8114610be057600080fd5b50565b600081359050610bf281610bcc565b92915050565b60008060408385031215610c0f57610c0e6106c8565b5b6000610c1d85828601610be3565b925050602083013567ffffffffffffffff811115610c3e57610c3d6106cd565b5b610c4a858286016109a4565b9150509250929050565b600081519050919050565b600082825260208201905092915050565b6000819050602082019050919050565b610c8981610bc0565b82525050565b6000604083016000830151610ca76000860182610c80565b5060208301518482036020860152610cbf8282610a8d565b9150508091505092915050565b6000610cd88383610c8f565b905092915050565b6000602082019050919050565b6000610cf882610c54565b610d028185610c5f565b935083602082028501610d1485610c70565b8060005b85811015610d505784840389528151610d318582610ccc565b9450610d3c83610ce0565b925060208a01995050600181019050610d18565b50829750879550505050505092915050565b6000606082019050610d776000830186610694565b610d846020830185610b96565b8181036040830152610d968184610ced565b9050949350505050565b600060208284031215610db657610db56106c8565b5b6000610dc4848285016107e7565b91505092915050565b610dd6816107be565b82525050565b6000602082019050610df16000830184610dcd565b92915050565b60006020820190508181036000830152610e118184610ced565b905092915050565b610e228161068a565b8114610e2d57600080fd5b50565b600081359050610e3f81610e19565b92915050565b600060208284031215610e5b57610e5a6106c8565b5b6000610e6984828501610e30565b91505092915050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052603260045260246000fd5b600081905092915050565b6000610eb782610a47565b610ec18185610ea1565b9350610ed1818560208601610a63565b80840191505092915050565b6000610ee98284610eac565b915081905092915050565b600082825260208201905092915050565b7f4d756c746963616c6c206167677265676174653a2063616c6c206661696c6564600082015250565b6000610f3b602083610ef4565b9150610f4682610f05565b602082019050919050565b60006020820190508181036000830152610f6a81610f2e565b9050919050565b7f4e487b7100000000000000000000000000000000000000000000000000000000600052601160045260246000fd5b6000610fab8261068a565b91507fffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff8203610fdd57610fdc610f71565b5b600182019050919050565b6000610ff38261068a565b9150610ffe8361068a565b925082820390508181111561101657611015610f71565b5b92915050565b7f4d756c746963616c6c32206167677265676174653a2063616c6c206661696c6560008201527f6400000000000000000000000000000000000000000000000000000000000000602082015250565b6000611078602183610ef4565b91506110838261101c565b604082019050919050565b600060208201905081810360008301526110a78161106b565b905091905056fea264697066735822122065ff5d84ac56a1dac443b9572e48402e647fef9c9384a2b0b89d6d81c070854864736f6c63430008130033",
}

// Multicall2ABI is the input ABI used to generate the binding from.
// Deprecated: Use Multicall2MetaData.ABI instead.
var Multicall2ABI = Multicall2MetaData.ABI

// Multicall2Bin is the compiled bytecode used for deploying new contracts.
// Deprecated: Use Multicall2MetaData.Bin instead.
var Multicall2Bin = Multicall2MetaData.Bin

// DeployMulticall2 deploys a new Ethereum contract, binding an instance of Multicall2 to it.
func DeployMulticall2(auth *bind.TransactOpts, backend bind.ContractBackend) (common.Address, *types.Transaction, *Multicall2, error) {
	parsed, err := Multicall2MetaData.GetAbi()
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	if parsed == nil {
		return common.Address{}, nil, nil, errors.New("GetABI returned nil")
	}

	address, tx, contract, err := bind.DeployContract(auth, *parsed, common.FromHex(Multicall2Bin), backend)
	if err != nil {
		return common.Address{}, nil, nil, err
	}
	return address, tx, &Multicall2{Multicall2Caller: Multicall2Caller{contract: contract}, Multicall2Transactor: Multicall2Transactor{contract: contract}, Multicall2Filterer: Multicall2Filterer{contract: contract}}, nil
}

// Multicall2 is an auto generated Go binding around an Ethereum contract.
type Multicall2 struct {
	Multicall2Caller     // Read-only binding to the contract
	Multicall2Transactor // Write-only binding to the contract
	Multicall2Filterer   // Log filterer for contract events
}

// Multicall2Caller is an auto generated read-only Go binding around an Ethereum contract.
type Multicall2Caller struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Multicall2Transactor is an auto generated write-only Go binding around an Ethereum contract.
type Multicall2Transactor struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Multicall2Filterer is an auto generated log filtering Go binding around an Ethereum contract events.
type Multicall2Filterer struct {
	contract *bind.BoundContract // Generic contract wrapper for the low level calls
}

// Multicall2Session is an auto generated Go binding around an Ethereum contract,
// with pre-set call and transact options.
type Multicall2Session struct {
	Contract     *Multicall2       // Generic contract binding to set the session for
	CallOpts     bind.CallOpts     // Call options to use throughout this session
	TransactOpts bind.TransactOpts // Transaction auth options to use throughout this session
}

// Multicall2CallerSession is an auto generated read-only Go binding around an Ethereum contract,
// with pre-set call options.
type Multicall2CallerSession struct {
	Contract *Multicall2Caller // Generic contract caller binding to set the session for
	CallOpts bind.CallOpts     // Call options to use throughout this session
}

// Multicall2TransactorSession is an auto generated write-only Go binding around an Ethereum contract,
// with pre-set transact options.
type Multicall2TransactorSession struct {
	Contract     *Multicall2Transactor // Generic contract transactor binding to set the session for
	TransactOpts bind.TransactOpts     // Transaction auth options to use throughout this session
}

// Multicall2Raw is an auto generated low-level Go binding around an Ethereum contract.
type Multicall2Raw struct {
	Contract *Multicall2 // Generic contract binding to access the raw methods on
}

// Multicall2CallerRaw is an auto generated low-level read-only Go binding around an Ethereum contract.
type Multicall2CallerRaw struct {
	Contract *Multicall2Caller // Generic read-only contract binding to access the raw methods on
}

// Multicall2TransactorRaw is an auto generated low-level write-only Go binding around an Ethereum contract.
type Multicall2TransactorRaw struct {
	Contract *Multicall2Transactor // Generic write-only contract binding to access the raw methods on
}

// NewMulticall2 creates a new instance of Multicall2, bound to a specific deployed contract.
func NewMulticall2(address common.Address, backend bind.ContractBackend) (*Multicall2, error) {
	contract, err := bindMulticall2(address, backend, backend, backend)
	if err != nil {
		return nil, err
	}
	return &Multicall2{Multicall2Caller: Multicall2Caller{contract: contract}, Multicall2Transactor: Multicall2Transactor{contract: contract}, Multicall2Filterer: Multicall2Filterer{contract: contract}}, nil
}

// NewMulticall2Caller creates a new read-only instance of Multicall2, bound to a specific deployed contract.
func NewMulticall2Caller(address common.Address, caller bind.ContractCaller) (*Multicall2Caller, error) {
	contract, err := bindMulticall2(address, caller, nil, nil)
	if err != nil {
		return nil, err
	}
	return &Multicall2Caller{contract: contract}, nil
}

// NewMulticall2Transactor creates a new write-only instance of Multicall2, bound to a specific deployed contract.
func NewMulticall2Transactor(address common.Address, transactor bind.ContractTransactor) (*Multicall2Transactor, error) {
	contract, err := bindMulticall2(address, nil, transactor, nil)
	if err != nil {
		return nil, err
	}
	return &Multicall2Transactor{contract: contract}, nil
}

// NewMulticall2Filterer creates a new log filterer instance of Multicall2, bound to a specific deployed contract.
func NewMulticall2Filterer(address common.Address, filterer bind.ContractFilterer) (*Multicall2Filterer, error) {
	contract, err := bindMulticall2(address, nil, nil, filterer)
	if err != nil {
		return nil, err
	}
	return &Multicall2Filterer{contract: contract}, nil
}

// bindMulticall2 binds a generic wrapper to an already deployed contract.
func bindMulticall2(address common.Address, caller bind.ContractCaller, transactor bind.ContractTransactor, filterer bind.ContractFilterer) (*bind.BoundContract, error) {
	parsed, err := Multicall2MetaData.GetAbi()
	if err != nil {
		return nil, err
	}
	return bind.NewBoundContract(address, *parsed, caller, transactor, filterer), nil
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Multicall2 *Multicall2Raw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Multicall2.Contract.Multicall2Caller.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Multicall2 *Multicall2Raw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Multicall2.Contract.Multicall2Transactor.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Multicall2 *Multicall2Raw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Multicall2.Contract.Multicall2Transactor.contract.Transact(opts, method, params...)
}

// Call invokes the (constant) contract method with params as input values and
// sets the output to result. The result type might be a single field for simple
// returns, a slice of interfaces for anonymous returns and a struct for named
// returns.
func (_Multicall2 *Multicall2CallerRaw) Call(opts *bind.CallOpts, result *[]interface{}, method string, params ...interface{}) error {
	return _Multicall2.Contract.contract.Call(opts, result, method, params...)
}

// Transfer initiates a plain transaction to move funds to the contract, calling
// its default method if one is available.
func (_Multicall2 *Multicall2TransactorRaw) Transfer(opts *bind.TransactOpts) (*types.Transaction, error) {
	return _Multicall2.Contract.contract.Transfer(opts)
}

// Transact invokes the (paid) contract method with params as input values.
func (_Multicall2 *Multicall2TransactorRaw) Transact(opts *bind.TransactOpts, method string, params ...interface{}) (*types.Transaction, error) {
	return _Multicall2.Contract.contract.Transact(opts, method, params...)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_Multicall2 *Multicall2Caller) GetBlockHash(opts *bind.CallOpts, blockNumber *big.Int) ([32]byte, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getBlockHash", blockNumber)

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_Multicall2 *Multicall2Session) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _Multicall2.Contract.GetBlockHash(&_Multicall2.CallOpts, blockNumber)
}

// GetBlockHash is a free data retrieval call binding the contract method 0xee82ac5e.
//
// Solidity: function getBlockHash(uint256 blockNumber) view returns(bytes32 blockHash)
func (_Multicall2 *Multicall2CallerSession) GetBlockHash(blockNumber *big.Int) ([32]byte, error) {
	return _Multicall2.Contract.GetBlockHash(&_Multicall2.CallOpts, blockNumber)
}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_Multicall2 *Multicall2Caller) GetBlockNumber(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getBlockNumber")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_Multicall2 *Multicall2Session) GetBlockNumber() (*big.Int, error) {
	return _Multicall2.Contract.GetBlockNumber(&_Multicall2.CallOpts)
}

// GetBlockNumber is a free data retrieval call binding the contract method 0x42cbb15c.
//
// Solidity: function getBlockNumber() view returns(uint256 blockNumber)
func (_Multicall2 *Multicall2CallerSession) GetBlockNumber() (*big.Int, error) {
	return _Multicall2.Contract.GetBlockNumber(&_Multicall2.CallOpts)
}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_Multicall2 *Multicall2Caller) GetCurrentBlockCoinbase(opts *bind.CallOpts) (common.Address, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getCurrentBlockCoinbase")

	if err != nil {
		return *new(common.Address), err
	}

	out0 := *abi.ConvertType(out[0], new(common.Address)).(*common.Address)

	return out0, err

}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_Multicall2 *Multicall2Session) GetCurrentBlockCoinbase() (common.Address, error) {
	return _Multicall2.Contract.GetCurrentBlockCoinbase(&_Multicall2.CallOpts)
}

// GetCurrentBlockCoinbase is a free data retrieval call binding the contract method 0xa8b0574e.
//
// Solidity: function getCurrentBlockCoinbase() view returns(address coinbase)
func (_Multicall2 *Multicall2CallerSession) GetCurrentBlockCoinbase() (common.Address, error) {
	return _Multicall2.Contract.GetCurrentBlockCoinbase(&_Multicall2.CallOpts)
}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_Multicall2 *Multicall2Caller) GetCurrentBlockDifficulty(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getCurrentBlockDifficulty")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_Multicall2 *Multicall2Session) GetCurrentBlockDifficulty() (*big.Int, error) {
	return _Multicall2.Contract.GetCurrentBlockDifficulty(&_Multicall2.CallOpts)
}

// GetCurrentBlockDifficulty is a free data retrieval call binding the contract method 0x72425d9d.
//
// Solidity: function getCurrentBlockDifficulty() view returns(uint256 difficulty)
func (_Multicall2 *Multicall2CallerSession) GetCurrentBlockDifficulty() (*big.Int, error) {
	return _Multicall2.Contract.GetCurrentBlockDifficulty(&_Multicall2.CallOpts)
}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_Multicall2 *Multicall2Caller) GetCurrentBlockGasLimit(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getCurrentBlockGasLimit")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_Multicall2 *Multicall2Session) GetCurrentBlockGasLimit() (*big.Int, error) {
	return _Multicall2.Contract.GetCurrentBlockGasLimit(&_Multicall2.CallOpts)
}

// GetCurrentBlockGasLimit is a free data retrieval call binding the contract method 0x86d516e8.
//
// Solidity: function getCurrentBlockGasLimit() view returns(uint256 gaslimit)
func (_Multicall2 *Multicall2CallerSession) GetCurrentBlockGasLimit() (*big.Int, error) {
	return _Multicall2.Contract.GetCurrentBlockGasLimit(&_Multicall2.CallOpts)
}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_Multicall2 *Multicall2Caller) GetCurrentBlockTimestamp(opts *bind.CallOpts) (*big.Int, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getCurrentBlockTimestamp")

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_Multicall2 *Multicall2Session) GetCurrentBlockTimestamp() (*big.Int, error) {
	return _Multicall2.Contract.GetCurrentBlockTimestamp(&_Multicall2.CallOpts)
}

// GetCurrentBlockTimestamp is a free data retrieval call binding the contract method 0x0f28c97d.
//
// Solidity: function getCurrentBlockTimestamp() view returns(uint256 timestamp)
func (_Multicall2 *Multicall2CallerSession) GetCurrentBlockTimestamp() (*big.Int, error) {
	return _Multicall2.Contract.GetCurrentBlockTimestamp(&_Multicall2.CallOpts)
}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_Multicall2 *Multicall2Caller) GetEthBalance(opts *bind.CallOpts, addr common.Address) (*big.Int, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getEthBalance", addr)

	if err != nil {
		return *new(*big.Int), err
	}

	out0 := *abi.ConvertType(out[0], new(*big.Int)).(**big.Int)

	return out0, err

}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_Multicall2 *Multicall2Session) GetEthBalance(addr common.Address) (*big.Int, error) {
	return _Multicall2.Contract.GetEthBalance(&_Multicall2.CallOpts, addr)
}

// GetEthBalance is a free data retrieval call binding the contract method 0x4d2301cc.
//
// Solidity: function getEthBalance(address addr) view returns(uint256 balance)
func (_Multicall2 *Multicall2CallerSession) GetEthBalance(addr common.Address) (*big.Int, error) {
	return _Multicall2.Contract.GetEthBalance(&_Multicall2.CallOpts, addr)
}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_Multicall2 *Multicall2Caller) GetLastBlockHash(opts *bind.CallOpts) ([32]byte, error) {
	var out []interface{}
	err := _Multicall2.contract.Call(opts, &out, "getLastBlockHash")

	if err != nil {
		return *new([32]byte), err
	}

	out0 := *abi.ConvertType(out[0], new([32]byte)).(*[32]byte)

	return out0, err

}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_Multicall2 *Multicall2Session) GetLastBlockHash() ([32]byte, error) {
	return _Multicall2.Contract.GetLastBlockHash(&_Multicall2.CallOpts)
}

// GetLastBlockHash is a free data retrieval call binding the contract method 0x27e86d6e.
//
// Solidity: function getLastBlockHash() view returns(bytes32 blockHash)
func (_Multicall2 *Multicall2CallerSession) GetLastBlockHash() ([32]byte, error) {
	return _Multicall2.Contract.GetLastBlockHash(&_Multicall2.CallOpts)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) returns(uint256 blockNumber, bytes[] returnData)
func (_Multicall2 *Multicall2Transactor) Aggregate(opts *bind.TransactOpts, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.contract.Transact(opts, "aggregate", calls)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) returns(uint256 blockNumber, bytes[] returnData)
func (_Multicall2 *Multicall2Session) Aggregate(calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.Aggregate(&_Multicall2.TransactOpts, calls)
}

// Aggregate is a paid mutator transaction binding the contract method 0x252dba42.
//
// Solidity: function aggregate((address,bytes)[] calls) returns(uint256 blockNumber, bytes[] returnData)
func (_Multicall2 *Multicall2TransactorSession) Aggregate(calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.Aggregate(&_Multicall2.TransactOpts, calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall2 *Multicall2Transactor) BlockAndAggregate(opts *bind.TransactOpts, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.contract.Transact(opts, "blockAndAggregate", calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall2 *Multicall2Session) BlockAndAggregate(calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.BlockAndAggregate(&_Multicall2.TransactOpts, calls)
}

// BlockAndAggregate is a paid mutator transaction binding the contract method 0xc3077fa9.
//
// Solidity: function blockAndAggregate((address,bytes)[] calls) returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall2 *Multicall2TransactorSession) BlockAndAggregate(calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.BlockAndAggregate(&_Multicall2.TransactOpts, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) returns((bool,bytes)[] returnData)
func (_Multicall2 *Multicall2Transactor) TryAggregate(opts *bind.TransactOpts, requireSuccess bool, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.contract.Transact(opts, "tryAggregate", requireSuccess, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) returns((bool,bytes)[] returnData)
func (_Multicall2 *Multicall2Session) TryAggregate(requireSuccess bool, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.TryAggregate(&_Multicall2.TransactOpts, requireSuccess, calls)
}

// TryAggregate is a paid mutator transaction binding the contract method 0xbce38bd7.
//
// Solidity: function tryAggregate(bool requireSuccess, (address,bytes)[] calls) returns((bool,bytes)[] returnData)
func (_Multicall2 *Multicall2TransactorSession) TryAggregate(requireSuccess bool, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.TryAggregate(&_Multicall2.TransactOpts, requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall2 *Multicall2Transactor) TryBlockAndAggregate(opts *bind.TransactOpts, requireSuccess bool, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.contract.Transact(opts, "tryBlockAndAggregate", requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall2 *Multicall2Session) TryBlockAndAggregate(requireSuccess bool, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.TryBlockAndAggregate(&_Multicall2.TransactOpts, requireSuccess, calls)
}

// TryBlockAndAggregate is a paid mutator transaction binding the contract method 0x399542e9.
//
// Solidity: function tryBlockAndAggregate(bool requireSuccess, (address,bytes)[] calls) returns(uint256 blockNumber, bytes32 blockHash, (bool,bytes)[] returnData)
func (_Multicall2 *Multicall2TransactorSession) TryBlockAndAggregate(requireSuccess bool, calls []Multicall2Call) (*types.Transaction, error) {
	return _Multicall2.Contract.TryBlockAndAggregate(&_Multicall2.TransactOpts, requireSuccess, calls)
}
