package live

import (
	"encoding/json"
	"math/big"

	ptracer "github.com/Chaintable/pipeline/tracer"
	ptypes "github.com/Chaintable/pipeline/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
)

type RpcTracer struct {
	tracer      ptracer.RPCTracer
	borTxIndex  uint
	borTxActive bool
}

func NewRpcTracer() RpcTracer {
	return RpcTracer{
		tracer: ptracer.RPCTracer{},
	}
}

func (t *RpcTracer) OnBlockStart(block *types.Block) {
	t.borTxIndex = borTxIndexForBlock(block)
	t.borTxActive = false
	t.tracer.OnBlockStart(block)
}

func (t *RpcTracer) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.tracer.OnTxStart(vm, tx, from)
}

func (t *RpcTracer) OnBorTxStart(txHash common.Hash) {
	//https://github.com/erigontech/erigon/blob/main/core/blockchain.go#L60
	tx := types.NewBorTransactionWithGasLimit(30_000_000)
	tx.SetHash(txHash)
	t.borTxActive = true
	t.tracer.OnTxStart(nil, tx, common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe"))
}

func (t *RpcTracer) OnTxEnd(receipt *types.Receipt, err error) {
	if t.borTxActive {
		if receipt != nil {
			receipt.TransactionIndex = t.borTxIndex
		}
		t.borTxActive = false
	}
	t.tracer.OnTxEnd(receipt, err)
}

func (t *RpcTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.tracer.OnEnter(depth, typ, from, to, input, gas, value)
}

func (t *RpcTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	t.tracer.OnExit(depth, output, gasUsed, err, reverted)
}

func (t *RpcTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	t.tracer.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
}

func (t *RpcTracer) OnLog(log *types.Log) {
	t.tracer.OnLog(log)
}

func (t *RpcTracer) GetResult() (json.RawMessage, error) {
	return t.tracer.GetResult()
}

func (t *RpcTracer) Stop(err error) {
	t.tracer.Stop(err)
}

func (t *RpcTracer) GetOutPut(originRoot common.Hash, root common.Hash, destructs map[common.Hash]struct{}, accounts map[common.Hash][]byte, storages map[common.Hash]map[common.Hash][]byte, codes map[common.Hash][]byte) *ptypes.DebankOutPut {
	return t.tracer.GetOutPut(originRoot, root, destructs, accounts, storages, codes)
}
