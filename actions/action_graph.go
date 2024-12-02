package actions

import (
	"evm-trackooor/shared"
	"evm-trackooor/utils"
	"fmt"
	"log"
	"math/big"
	"os"
	"os/exec"
	"slices"
	"sync"

	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/ethereum/go-ethereum/common"
)

func (actionInfo) InfoGraph() actionInfo {
	name := "Graph"
	overview := `Graphs transactions, native ETH and ERC20 transfers, originating from/to specific addresses.`

	description := `User specifies addresses to track, and what to graph, such as transactions and ERC20 transfers. ` +
		`Other addresses that interact with tracked addresses also become tracked, to visualize flow of txs/funds.

For example, let -> denote a transfer. You want to track transfers from A. A -> B, causes B to be tracked.` +
		`If B -> C, then the graph will show A -> B -> C.
		
However, due to the inherent design of the EVM trackooor, if A -> B -> C occurs all in the same block, then C may not be tracked.

If graphing transactions, will attempt to decode transaction data to show the function called and function arguments. Decoding successfully will require function signature ABI, check the README for adding function signature data.

The EVM trackooor design isn't suited towards graphing, but this is an example of how you could graph data from the blockchain.
Note that this action is still in development and may contain bugs and undercoverage of transfers/txs.

To graph, graphviz is used, so graphviz and the 'dot' command must be installed on the system.`

	options := `"graph-transactions" - Whether or not to graph transactions
"graph-internal-txs" - Whether or not to graph internal transactions (this will require your RPC to support Debug API)
"graph-eth-transfers" - Whether or not to graph native ETH transfers (transactions with value > 0)
"eth-transfer-threshold" - When graphing ETH transfers, the minimum value of the transfer in wei for transfer to be graphed
"graph-erc20-transfers" - Whether or not to graph ERC20 token transfers
"erc20-tokens" - Array of ERC20 token addresses, which are the ERC20 tokens you want to graph
"graph-incoming" - Whether or not to graph incoming txs/transfers to tracked addresses (otherwise only outgoing txs/transfers from tracked addresses will be recorded)
"max-depth" - Max graph depth from the originally specified tracked addresses.
"output-filepath" - Filepath to output the PNG graph image.
`

	example := `"Graph": {
    "addresses":{
        "0x77aFC774c38D6A712e1A1F5Ea7c88Fe14BFA10F6":{
            "name":"Fake_Phishing339532"
        }
    },
    "options":{
        "graph-transactions":false,
        "graph-eth-transfers":true,
        "graph-erc20-transfers":true,
        "graph-incoming":true,
        "erc20-tokens":[
            "0xdAC17F958D2ee523a2206206994597C13D831ec7",
            "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
            "0xD4F4D0a10BcaE123bB6655E8Fe93a30d01eEbD04"
        ],
        "eth-transfer-threshold":"1000000000000000000",
        "max-depth":2,
        "output-filepath":"./graph.png"
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

// logging
var graphLog *log.Logger

// options
var graphOptions struct {
	graphTransactions    bool     // graph all txs. setting this to true ignores graphEthTransfers bool // only graph txs that transferred eth.
	graphEthTransfers    bool     // only graph txs that transferred eth.
	ethTransferThreshold *big.Int // only graph transfers with more than this amount (in wei)
	graphERC20Transfers  bool     // graph erc20 token transfers
	graphIncoming        bool     // graph txs/transfers incoming into tracked addresses (usually only outgoing is graphed)
	graphMaxDepth        int

	graphInternalTxs bool // graph internal txs

	// data options
	outputFilepath string
}

// graph
var gMutex sync.Mutex
var g graph.Graph[string, common.Address]
var generateGraphMutex sync.Mutex
var graphFinishedMutex sync.WaitGroup

// addresses to graph
var graphAddresses []common.Address

// originally supplied addresses
var originalAddresses []common.Address

// erc20 tokens to graph
type erc20Token struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

var ETH common.Address //  we will treat native ETH as an ERC20 token with address 0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee
var graphERC20Tokens map[common.Address]erc20Token

func (p action) InitGraph() {
	// init maps
	graphERC20Tokens = make(map[common.Address]erc20Token)

	// init logger
	graphLog = shared.TimeLogger("[Graph] ")

	// init graph data
	g = graph.New(common.Address.Hex, graph.Directed())

	// set custom options
	setGraphOptions(p.o.CustomOptions)

	// if tracking requires tx
	// given addresses are what we start tracking from
	if graphOptions.graphEthTransfers {
		for _, address := range p.o.Addresses {
			addTxAddressAction(address, graphHandleTx)
		}
	}

	// if tracking erc20 transfers, listen for transfer event sig
	if graphOptions.graphERC20Transfers {
		for token, _ := range graphERC20Tokens {
			addAddressEventSigAction(
				token,
				"Transfer(address,address,uint256)",
				graphHandleERC20Transfer,
			)
		}
	}

	// init eth const
	// intentionally done after erc20 tokens are tracked so 0xeee... doesnt get tracked
	ETH = common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee")
	graphERC20Tokens[ETH] = erc20Token{
		Name:     "Native ETH",
		Symbol:   "ETH",
		Decimals: 18,
	}

	// add known vertexes to graph (the addresses we are tracking)
	graphAddresses = p.o.Addresses
	originalAddresses = p.o.Addresses
	for _, address := range graphAddresses {
		// highlight with green
		g.AddVertex(
			address,
			graph.VertexAttribute("label", getAddressName(address)),
			graph.VertexAttribute("colorscheme", "greens3"),
			graph.VertexAttribute("style", "filled"),
			graph.VertexAttribute("color", "2"),
			graph.VertexAttribute("fillcolor", "1"),
		)
	}
}

func (p action) FinishedGraph() {
	graphLog.Printf("Waiting for processing funcs to finish\n")
	graphFinishedMutex.Wait()

	graphLog.Printf("Generating graph to %v\n", graphOptions.outputFilepath)
	generateGraphImage(graphOptions.outputFilepath)
}

func setGraphOptions(p map[string]interface{}) {
	if v, ok := p["graph-transactions"]; ok {
		graphOptions.graphTransactions = v.(bool)
	}
	if v, ok := p["graph-eth-transfers"]; ok {
		graphOptions.graphEthTransfers = v.(bool)
	}
	if v, ok := p["eth-transfer-threshold"]; ok {
		e := big.NewInt(0)
		e, ok2 := e.SetString(v.(string), 10)
		if !ok2 {
			graphLog.Fatalf("Invalid \"eth-transfer-threshold\" value: %v\n", e)
		}
		graphOptions.ethTransferThreshold = e
	} else {
		graphOptions.ethTransferThreshold = big.NewInt(1) // default min 1 wei
	}
	if v, ok := p["graph-erc20-transfers"]; ok {
		graphOptions.graphERC20Transfers = v.(bool)
		if graphOptions.graphERC20Transfers {
			if v, ok := p["erc20-tokens"]; ok {
				addressInterfaces := v.([]interface{})
				for _, addressInterface := range addressInterfaces {
					erc20Address := common.HexToAddress(addressInterface.(string))
					// fetch erc20 metadata (name, symbol, decimals)
					tokenInfo := shared.RetrieveERC20Info(erc20Address)
					erc20TokenInfo := erc20Token{
						Name:     tokenInfo.Name,
						Symbol:   tokenInfo.Symbol,
						Decimals: tokenInfo.Decimals,
					}
					graphERC20Tokens[erc20Address] = erc20TokenInfo
				}
			}
		}
	}
	if v, ok := p["graph-incoming"]; ok {
		graphOptions.graphIncoming = v.(bool)
	}

	if v, ok := p["max-depth"]; ok {
		graphOptions.graphMaxDepth = int(v.(float64))
	}

	if v, ok := p["graph-internal-txs"]; ok {
		graphOptions.graphInternalTxs = v.(bool)
	}

	if v, ok := p["output-filepath"]; ok {
		graphOptions.outputFilepath = v.(string)
	} else {
		graphLog.Fatal("\"output-filepath\" not specified!")
	}

	// output options
	graphLog.Printf("Graph transactions: %v\n", graphOptions.graphTransactions)
	// all eth transfers are transactions. if graphing transactions, eth transfers will be graphed
	if !graphOptions.graphTransactions {
		graphLog.Printf("Graph eth transfers: %v\n", graphOptions.graphEthTransfers)
		graphLog.Printf("Eth transfer threshold: %v wei\n", graphOptions.ethTransferThreshold)
	}
	graphLog.Printf("Graph ERC20 transfers: %v\n", graphOptions.graphERC20Transfers)
	graphLog.Printf("Graph incoming txs/transfers: %v\n", graphOptions.graphIncoming)
	graphLog.Printf("Graph internal txs: %v\n", graphOptions.graphInternalTxs)
	graphLog.Printf("Graph output filepath: %v\n", graphOptions.outputFilepath)
	graphLog.Printf("Max depth (from provided addresses): %v\n", graphOptions.graphMaxDepth)
}

func recursivelyGraphTraceResult(tr shared.TraceResult, blockNum *big.Int) {
	switch tr.Type {
	case "CALL", "DELEGATECALL":
		// graph as tx
		// fmt.Printf("CALL OR DELEGATE CALL\n")

		from := tr.From
		to := tr.To
		txData := tr.Input
		if len(txData) >= 4 {
			funcSelectorHex := "0x" + common.Bytes2Hex(txData[:4])
			if funcAbi, ok := shared.FuncSigs[funcSelectorHex]; ok {
				funcSig, funcArgs, err := shared.DecodeTransaction(funcSelectorHex, txData, funcAbi)
				if err != nil {
					graphLog.Printf("Failed to decode input %v, ignoring\n", txData)
				}
				ok := graphTx(from, to, funcSig, funcArgs, blockNum, true)
				if !ok {
					graphLog.Fatalf("Failed to graph from: %v to: %v\n", from, to)
				}
			} else {
				graphLog.Printf("Don't recognise func selector %v\n", funcSelectorHex)
				ok := graphTx(from, to, funcSelectorHex, make(map[string]interface{}), blockNum, true)
				if !ok {
					graphLog.Fatalf("Failed to graph from: %v to: %v\n", from, to)
				}
			}
		} else {
			graphLog.Printf("Input len too short, no func selector %v\n", txData)
			ok := graphTx(from, to, string(txData), make(map[string]interface{}), blockNum, true)
			if !ok {
				graphLog.Fatalf("Failed to graph from: %v to: %v\n", from, to)
			}
		}

		// recurse
		for _, _tr := range tr.Calls {
			recursivelyGraphTraceResult(_tr, blockNum)
		}
	case "CREATE", "CREATE2":
		// fmt.Printf("CREATE OR CREATE2\n")

		// graph deployment
		contract := tr.To
		graphTransfer(ETH, tr.From, contract, tr.Value, true, blockNum, true)

		// recurse
		for _, _tr := range tr.Calls {
			recursivelyGraphTraceResult(_tr, blockNum)
		}
	}
}

func graphHandleTx(p ActionTxData) {
	graphFinishedMutex.Add(1)
	defer graphFinishedMutex.Done()

	if graphOptions.graphInternalTxs {
		traceResult, _ := shared.Debug_traceTransaction(p.Transaction.Hash())
		recursivelyGraphTraceResult(traceResult, p.Block.Number())
		// 	for _, tr := range traceResult.Calls {
		// 	recursivelyGraphTraceResult(tr, p.Block.Number())
		// }
		return
	}

	// special case if tx is deployment
	if utils.IsDeploymentTx(p.Transaction) {
		deployer := *p.From
		deployedContract, _ := shared.GetDeployedContractAddress(p.Transaction)
		value := p.Transaction.Value()
		success := graphTransfer(ETH, deployer, deployedContract, value, true, p.Block.Number(), false)
		if !success {
			return
		}
	} else {
		// otherwise, add to graph as normal
		from := *p.From
		to := *p.To
		value := p.Transaction.Value()
		ethThreshold := graphOptions.ethTransferThreshold
		// unless tracking incoming txs, we are only interested in txs that originate from
		// addresses we are tracking
		if !graphOptions.graphIncoming && !slices.Contains(graphAddresses, from) {
			return
		}

		if graphOptions.graphTransactions {
			txData := p.Transaction.Data()
			blockNum := p.Block.Number()
			if len(txData) >= 4 {
				funcSelectorHex := "0x" + common.Bytes2Hex(txData[:4])
				if funcAbi, ok := shared.FuncSigs[funcSelectorHex]; ok {
					funcSig, funcArgs, err := shared.DecodeTransaction(funcSelectorHex, txData, funcAbi)
					if err != nil {
						graphLog.Printf("Failed to decode transaction %v, ignoring\n", p.Transaction.Hash())
					}
					ok := graphTx(from, to, funcSig, funcArgs, p.Block.Number(), false)
					if !ok {
						graphLog.Printf("Failed to graph from: %v to: %v\n", from, to)
					}
				} else {
					graphLog.Printf("Don't recognise func selector %v\n", funcSelectorHex)
					ok := graphTx(from, to, funcSelectorHex, make(map[string]interface{}), blockNum, false)
					if !ok {
						graphLog.Printf("Failed to graph from: %v to: %v\n", from, to)
					}
				}
			} else {
				graphLog.Printf("Tx data len too short, no func selector %v\n", txData)
				ok := graphTx(from, to, string(txData), make(map[string]interface{}), blockNum, false)
				if !ok {
					graphLog.Printf("Failed to graph from: %v to: %v\n", from, to)
				}
			}
		}

		// if we're graphing ETH transfers, check if eth was transferred
		// otherwise check if we're graphing all txs
		// value > 0
		if graphOptions.graphTransactions || value.Cmp(ethThreshold) >= 0 {
			success := graphTransfer(ETH, from, to, value, false, p.Block.Number(), false)
			if !success {
				return
			}
			// track both entities
			// (doesnt matter if already tracked)
			addTxAddressAction(from, graphHandleTx)
			addTxAddressAction(to, graphHandleTx)
		}
	}
}

func graphHandleERC20Transfer(p ActionEventData) {
	graphFinishedMutex.Add(1)
	defer graphFinishedMutex.Done()

	to := p.DecodedTopics["to"].(common.Address)
	from := p.DecodedTopics["from"].(common.Address)
	value := p.DecodedData["value"].(*big.Int)

	tokenAddress := p.EventLog.Address

	// make sure either `to` or `from` is an address we are graphing
	// if not, dont graph
	if !slices.Contains(graphAddresses, to) && !slices.Contains(graphAddresses, from) {
		return
	}

	success := graphTransfer(tokenAddress, from, to, value, false, big.NewInt(int64(p.EventLog.BlockNumber)), false)
	if !success {
		return
	}

	// track the addresses (if not already tracked)
	if !slices.Contains(graphAddresses, to) {
		graphAddresses = append(graphAddresses, to)
	}
	if !slices.Contains(graphAddresses, from) {
		graphAddresses = append(graphAddresses, from)
	}
}

func getAddressVertexAttributes(address common.Address, wasDeployed bool, blockNum *big.Int) []func(*graph.VertexProperties) {
	var vertexAttributes []func(*graph.VertexProperties)

	vertexAttributes = append(vertexAttributes, graph.VertexAttribute("label", getAddressName(address)))

	if wasDeployed {
		vertexAttributes = append(vertexAttributes,
			graph.VertexAttribute("colorscheme", "blues5"),
			graph.VertexAttribute("style", "filled"),
			graph.VertexAttribute("color", "4"),
			graph.VertexAttribute("fillcolor", "3"),
		)
		return vertexAttributes
	}
	addressType := shared.DetermineAddressType(address, blockNum)
	switch addressType {
	case shared.ContractTx:
		vertexAttributes = append(vertexAttributes,
			graph.VertexAttribute("colorscheme", "blues5"),
			graph.VertexAttribute("style", "filled"),
			graph.VertexAttribute("color", "2"),
			graph.VertexAttribute("fillcolor", "1"),
		)
	case shared.RegularTx:
		// no colour for EOAs
	}
	return vertexAttributes
}

func minDepthFromNodes(currentNode common.Address, nodes []common.Address) int {
	var depths []int
	// fmt.Printf("current node: %v\n", currentNode)
	// fmt.Printf("nodes: %v\n", nodes)
	for _, node := range nodes {
		path, err := graph.ShortestPath(g, currentNode.Hex(), node.Hex())
		depth := len(path) - 1
		if err != nil {
			path, err := graph.ShortestPath(g, node.Hex(), currentNode.Hex())
			depth = len(path) - 1
			if err != nil {
				graphLog.Printf("minDepthFromNodes err: %v from: %v to: %v\n", err, node, currentNode)
				continue
			}
		}
		depths = append(depths, depth)
	}
	// fmt.Printf("MIN DEPTH: %v\n", slices.Min(depths))
	if len(depths) == 0 {
		return graphOptions.graphMaxDepth + 1
	}
	return slices.Min(depths)
}

func addEdgeLabel(from common.Address, to common.Address) {
	edge, _ := g.Edge(from.Hex(), to.Hex())
	labelStr := ""
	properties := edge.Properties.Data.(map[string]interface{})

	// transfers
	values := properties["transfer"].(map[common.Address]*big.Int)
	for tokenAddress, value := range values {
		tokenInfo := graphERC20Tokens[tokenAddress]
		decimals := tokenInfo.Decimals
		symbol := tokenInfo.Symbol
		if labelStr == "" {
			labelStr = utils.FormatDecimals(value, decimals) + " " + symbol
		} else {
			labelStr = labelStr + " + " + utils.FormatDecimals(value, decimals) + " " + symbol
		}
	}

	// txs
	txs := properties["tx"].(map[string][]interface{})

	for funcSig, funcArgs := range txs {
		for _, funcArgInterface := range funcArgs {
			labelStr = fmt.Sprintf("%v\\n%v", labelStr, funcSig)
			funcArgMap := funcArgInterface.(map[string]interface{})
			for funcParam, valueInterface := range funcArgMap {
				value, err := shared.BytesToHex(valueInterface)
				var vs string
				if err == nil {
					vs = value
				} else {
					vs = fmt.Sprintf("%v", valueInterface)
				}

				if len(vs) > 50 { // truncate if too long
					vs = fmt.Sprintf("%v... (truncated)", vs[:50])
				}
				labelStr = fmt.Sprintf("%v\\l    %v: %v", labelStr, funcParam, vs) // \l left aligns label line
			}
		}
	}
	labelStr = labelStr + "\\l"

	edge.Properties.Attributes["label"] = labelStr

	graphLog.Printf(
		"Edge from %v to %v label %v\n",
		utils.ShortenAddress(from),
		utils.ShortenAddress(to),
		labelStr,
	)
}

// add tx with funcsig and args to graph
// returns whether or not successfully added
func graphTx(from common.Address, to common.Address, funcSig string, funcArgs map[string]interface{}, blockNum *big.Int, ignoreDepth bool) bool {
	gMutex.Lock()
	defer gMutex.Unlock()

	// get vertex attributes
	fromVertexAttributes := getAddressVertexAttributes(from, false, blockNum)
	toVertexAttributes := getAddressVertexAttributes(to, false, blockNum)

	// add vertexes
	toWasAdded := true
	fromWasAdded := true
	err := g.AddVertex(to, toVertexAttributes...)
	if err == graph.ErrVertexAlreadyExists {
		toWasAdded = false
	}
	err = g.AddVertex(from, fromVertexAttributes...)
	if err == graph.ErrVertexAlreadyExists {
		fromWasAdded = false
	}

	// make edge between vertex, label with token value
	// if edge already exists, make with updated token value
	if edge, err := g.Edge(from.Hex(), to.Hex()); err == graph.ErrEdgeNotFound {
		// edge doesnt already exist
		// init
		properties := make(map[string]interface{})
		properties["tx"] = make(map[string][]interface{})
		properties["transfer"] = make(map[common.Address]*big.Int)

		// add to properties
		txs := properties["tx"].(map[string][]interface{})
		txs[funcSig] = append(txs[funcSig], funcArgs)
		properties["tx"] = txs
		g.AddEdge(
			from.Hex(),
			to.Hex(),
			graph.EdgeData(properties),
		)

		if !ignoreDepth {
			// if vertexes were newly added, make sure they're not too deep from original addresses
			if toWasAdded {
				depth := minDepthFromNodes(to, originalAddresses)
				if depth > graphOptions.graphMaxDepth {
					// too deep, remove edge then vertex
					g.RemoveEdge(from.Hex(), to.Hex())
					g.RemoveVertex(to.Hex())
					return false
				}
			}
			if fromWasAdded {
				depth := minDepthFromNodes(from, originalAddresses)
				if depth > graphOptions.graphMaxDepth {
					// too deep, remove edge then vertex
					g.RemoveEdge(from.Hex(), to.Hex())
					g.RemoveVertex(from.Hex())
					return false
				}
			}
		}
	} else {
		// edge already exists
		// update edge data
		properties := edge.Properties.Data.(map[string]interface{})
		txs := properties["tx"].(map[string][]interface{})

		txs[funcSig] = append(txs[funcSig], funcArgs)

		properties["tx"] = txs
		g.UpdateEdge(from.Hex(), to.Hex(), graph.EdgeData(properties))
	}

	// add edge label
	addEdgeLabel(from, to)

	return true
}

// returns true if transfer was graphed, false if transfer was not graphed due to e.g. too deep
func graphTransfer(tokenAddress common.Address, from common.Address, to common.Address, value *big.Int, isDeployment bool, blockNum *big.Int, ignoreDepth bool) bool {
	gMutex.Lock()
	defer gMutex.Unlock()
	// as there can be multiple erc20 tokens, we will use a mapping token address to value
	// to store the value of erc20 transfers
	// also, we will treat native ETH as a token with address 0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee

	// get vertex attributes
	fromVertexAttributes := getAddressVertexAttributes(from, false, blockNum)
	toVertexAttributes := getAddressVertexAttributes(to, isDeployment, blockNum)

	// add vertexes
	toWasAdded := true
	fromWasAdded := true
	err := g.AddVertex(to, toVertexAttributes...)
	if err == graph.ErrVertexAlreadyExists {
		toWasAdded = false
	}
	err = g.AddVertex(from, fromVertexAttributes...)
	if err == graph.ErrVertexAlreadyExists {
		fromWasAdded = false
	}

	// make edge between vertex, label with token value
	// if edge already exists, make with updated token value
	if edge, err := g.Edge(from.Hex(), to.Hex()); err == graph.ErrEdgeNotFound {
		// edge doesnt already exist
		// init
		properties := make(map[string]interface{})
		properties["tx"] = make(map[string][]interface{})
		properties["transfer"] = make(map[common.Address]*big.Int)

		// add to properties
		values := properties["transfer"].(map[common.Address]*big.Int)
		values[tokenAddress] = value
		properties["transfer"] = values
		g.AddEdge(
			from.Hex(),
			to.Hex(),
			graph.EdgeData(properties),
		)

		if !ignoreDepth {
			// if vertexes were newly added, make sure they're not too deep from original addresses
			if toWasAdded {
				depth := minDepthFromNodes(to, originalAddresses)
				if depth > graphOptions.graphMaxDepth {
					// too deep, remove edge then vertex
					g.RemoveEdge(from.Hex(), to.Hex())
					g.RemoveVertex(to.Hex())
					return false
				}
			}
			if fromWasAdded {
				depth := minDepthFromNodes(from, originalAddresses)
				if depth > graphOptions.graphMaxDepth {
					// too deep, remove edge then vertex
					g.RemoveEdge(from.Hex(), to.Hex())
					g.RemoveVertex(from.Hex())
					return false
				}
			}
		}

	} else {
		// edge already exists
		// update edge data
		properties := edge.Properties.Data.(map[string]interface{})
		values := properties["transfer"].(map[common.Address]*big.Int)
		if v, ok := values[tokenAddress]; ok {
			values[tokenAddress] = big.NewInt(0).Add(v, value)
		} else {
			values[tokenAddress] = value
		}
		properties["transfer"] = values
		g.UpdateEdge(from.Hex(), to.Hex(), graph.EdgeData(properties))
	}

	// add edge label
	addEdgeLabel(from, to)
	return true
}

func generateGraphImage(outputFilepath string) {
	generateGraphMutex.Lock()
	// convert graph to png image
	graphFilename := "/tmp/tempgraph.gv"
	file, _ := os.Create(graphFilename)
	// really, really hacky way of getting labels to align left
	// _ = draw.DOT(g, file, draw.GraphAttribute("node [labeljust=l];//", ""))
	_ = draw.DOT(g, file)

	cmd := exec.Command("dot", "-Tpng", fmt.Sprintf("-o%v", outputFilepath), graphFilename)
	stdout, err := cmd.Output()
	if err != nil {
		graphLog.Printf("Error when generating graph: %v stdout: %v\n", err, stdout)
		generateGraphMutex.Unlock()
		return
	}

	graphLog.Printf("Graph generated to %v\n", outputFilepath)

	generateGraphMutex.Unlock()
}

func getAddressName(address common.Address) string {
	// shortened address label
	label := utils.ShortenAddress(address)
	// or use name if address has a name
	if properties, ok := shared.Options.AddressProperties[address]; ok {
		if name, ok := properties["name"]; ok {
			label = name.(string)
		}
	}
	return label
}
