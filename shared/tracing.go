package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

// debug_traceTransaction code

type TraceResult struct {
	From    common.Address `json:"from"`
	To      common.Address `json:"to"`
	Gas     uint64         `json:"gas"`
	GasUsed uint64         `json:"gasUsed"`
	Input   []byte         `json:"input"`
	Output  []byte         `json:"output"`
	Value   *big.Int       `json:"value"`
	Type    string         `json:"type"`
	Calls   []TraceResult  `json:"calls"`
}

func (t *TraceResult) UnmarshalJSON(input []byte) error {
	type traceResultTmp struct {
		From    common.Address `json:"from"`
		To      common.Address `json:"to"`
		Gas     hexutil.Uint64 `json:"gas"`
		GasUsed hexutil.Uint64 `json:"gasUsed"`
		Input   hexutil.Bytes  `json:"input"`
		Output  hexutil.Bytes  `json:"output"`
		Value   *hexutil.Big   `json:"value"`
		Type    string         `json:"type"`
		Calls   []TraceResult  `json:"calls"`
	}

	var dec traceResultTmp
	if err := json.Unmarshal(input, &dec); err != nil {
		return err
	}

	t.From = dec.From
	t.To = dec.To
	t.Gas = uint64(dec.Gas)
	t.GasUsed = uint64(dec.GasUsed)
	t.Input = dec.Input
	t.Output = dec.Output
	t.Value = (*big.Int)(dec.Value)
	t.Type = dec.Type
	t.Calls = dec.Calls

	return nil
}

func Debug_traceTransaction(txHash common.Hash) (TraceResult, error) {
	tracer := make(map[string]interface{})
	tracer["tracer"] = "callTracer"

	var raw json.RawMessage
	Client.Client().CallContext(
		context.Background(),
		&raw,
		"debug_traceTransaction",
		txHash.Hex(),
		tracer,
	)

	var tResult TraceResult
	if err := json.Unmarshal(raw, &tResult); err != nil {
		// fmt.Printf("Debug_traceTransaction err txHash: %v err: %v\n", txHash, err)
		return TraceResult{}, fmt.Errorf("err: %v raw: %v\n", err, raw)
		// return Debug_traceTransaction(txHash)
	}
	return tResult, nil
}

// trace_transaction and trace_replayBlockTransactions

type Action struct {
	From     common.Address `json:"from"`
	To       common.Address `json:"to,omitempty"`
	CallType string         `json:"callType,omitempty"`
	Gas      hexutil.Uint64 `json:"gas"`
	Input    hexutil.Bytes  `json:"input,omitempty"`
	Init     hexutil.Bytes  `json:"init,omitempty"`
	Value    *hexutil.Big   `json:"value"`
}

type Result struct {
	GasUsed string         `json:"gasUsed"`
	Output  hexutil.Bytes  `json:"output,omitempty"`
	Address common.Address `json:"address,omitempty"`
	Code    hexutil.Bytes  `json:"code,omitempty"`
}

type Trace struct {
	Action              Action `json:"action"`
	BlockHash           string `json:"blockHash"`
	BlockNumber         int    `json:"blockNumber"`
	Result              Result `json:"result"`
	Subtraces           int    `json:"subtraces"`
	TraceAddress        []int  `json:"traceAddress"`
	TransactionHash     string `json:"transactionHash"`
	TransactionPosition int    `json:"transactionPosition"`
	Type                string `json:"type"`
}

func Trace_transaction(txHash common.Hash) []Trace {
	var raw json.RawMessage
	Client.Client().CallContext(
		context.Background(),
		&raw,
		"trace_transaction",
		txHash.Hex(),
	)

	// fmt.Printf("raw: %v\n", string(raw))

	var result []Trace
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		fmt.Printf("Trace_transaction err txHash: %v err: %v\n", txHash, err)
		return nil
		// return Trace_transaction(txHash)
	}

	return result
}

type BlockTrace struct {
	Output    hexutil.Bytes `json:"output"`
	StateDiff any           `json:"stateDiff"`
	Traces    []Trace       `json:"trace"`
}

// traceType is one or more of "trace", "stateDiff"
func Trace_replayBlockTransactions(blockNum uint64, traceType []string) ([]BlockTrace, error) {

	var raw json.RawMessage
	Client.Client().CallContext(
		context.Background(),
		&raw,
		"trace_replayBlockTransactions",
		blockNum, traceType, // params
	)

	// fmt.Printf("raw: %v\n", string(raw))

	var result []BlockTrace
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// fmt.Printf("Trace_replayBlockTransactions err blockNum: %v err: %v raw: %v\n", blockNum, err, raw)
		// return nil
		return nil, fmt.Errorf("Trace_replayBlockTransactions err blockNum: %v err: %v raw: %v", blockNum, err, raw)
	}

	return result, nil
}

type CallParams struct {
	From     common.Address `json:"from,omitempty"`
	To       common.Address `json:"to,omitempty"`
	Gas      uint64         `json:"gas,omitempty"`
	GasPrice uint64         `json:"gasPrice,omitempty"`
	Value    *hexutil.Big   `json:"value,omitempty"`
	Data     hexutil.Bytes  `json:"data,omitempty"`
}

type TraceCallResult struct {
	Output    string                        `json:"output"`
	StateDiff map[common.Address]StateEntry `json:"stateDiff"`
	Trace     []TraceEntry                  `json:"trace"`
	VmTrace   interface{}                   `json:"vmTrace"` // Use interface{} for null or future extensibility
}

type StateEntry struct {
	// these can either be "=" indicating nothing changed
	// or a json struct indicating whats changed
	Balance json.RawMessage `json:"balance"`
	Code    json.RawMessage `json:"code"`
	Nonce   json.RawMessage `json:"nonce,omitempty"`
	Storage json.RawMessage `json:"storage,omitempty"`
}

type TraceEntry struct {
	Action       TraceAction      `json:"action"`
	Result       *CallTraceResult `json:"result,omitempty"`
	Subtraces    int              `json:"subtraces"`
	TraceAddress []int            `json:"traceAddress"`
	Type         string           `json:"type"`
	Error        string           `json:"error,omitempty"`
}

type TraceAction struct {
	From     string         `json:"from"`
	CallType string         `json:"callType"`
	Gas      hexutil.Uint64 `json:"gas"`
	Input    string         `json:"input"`
	To       string         `json:"to"`
	Value    *hexutil.Big   `json:"value"`
}

type CallTraceResult struct {
	GasUsed hexutil.Uint64 `json:"gasUsed"`
	Output  string         `json:"output"`
}

// referencing https://docs.alchemy.com/reference/trace-call
// traceType - Type of trace, one or more of: "trace", "stateDiff"
func Trace_call(callParams CallParams, traceType []string) (TraceCallResult, error) {
	var raw json.RawMessage

	params := make(map[string]interface{})

	jsonData, err := json.Marshal(callParams)
	if err != nil {
		log.Fatalf("Error marshaling to JSON: %v", err)
	}

	err = json.Unmarshal(jsonData, &params)
	if err != nil {
		log.Fatalf("Error unmarshaling JSON to map: %v", err)
	}

	// fmt.Printf("params: %v\n", params)

	err = Client.Client().CallContext(
		context.Background(),
		&raw,
		"trace_call",
		params, // params
		traceType,
	)
	if err != nil {
		fmt.Printf("CallContext err: %v\n", err)
	}

	// fmt.Printf("raw: %v\n", string(raw))

	var result TraceCallResult
	if err := json.Unmarshal([]byte(raw), &result); err != nil {
		// fmt.Printf("Trace_replayBlockTransactions err blockNum: %v err: %v raw: %v\n", blockNum, err, raw)
		// return nil
		return TraceCallResult{}, fmt.Errorf("Trace_call err: %v params: %v traceType: %v", err, callParams, traceType)
	}

	return result, nil
}
