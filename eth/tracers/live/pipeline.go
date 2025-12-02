package live

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/Chaintable/pipeline/tracer"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
)

// 需要上传3种data
// 1. block
// 2. state diff
// 3. block file

func init() {
	tracers.LiveDirectory.Register("pipeline", NewPipelineTracer)
}

func NewPipelineTracer(cfg json.RawMessage) (*tracing.Hooks, error) {
	t, err := NewNativePipelineTracer(cfg)
	if err != nil {
		return nil, err
	}
	return &tracing.Hooks{
		OnBlockchainInit: t.OnBlockchainInit,
		OnClose:          t.OnClose,
		OnBlockStart:     t.OnBlockStart,
		OnTxStart:        t.OnTxStart,
		OnTxEnd:          t.OnTxEnd,
		OnEnter:          t.OnEnter,
		OnExit:           t.OnExit,
		OnLog:            t.OnLog,
		OnOpcode:         t.OnOpcode,
		OnGenesisBlock:   t.OnGenesisBlock,
		OnCommit:         t.OnCommit,
		OnBorTxStart:     t.OnBorTxStart,
	}, nil
}

// 需要上传3种data
// 1. block
// 2. state diff
// 3. block file

type PipelineTracer struct {
	pipelineTracer *tracer.PipelineTracer
}

type pipelineTracerConfig struct {
	Region               string   `json:"region"`
	NodeXBucket          string   `json:"node_x_bucket"`
	ChainTableBucket     string   `json:"chain_table_bucket"`
	Brokers              []string `json:"brokers"`
	Topic                string   `json:"topic"`
	S3TempDir            string   `json:"s3_temp_dir"`
	IsBackup             *bool    `json:"is_backup"` // nil = auto (use etcd), false = leader in fixed mode, true = backup in fixed mode
	EnablePreStateTracer bool     `json:"enable_prestate_tracer"`

	// Auto failover configurations
	EtcdEndpoints []string `json:"etcd_endpoints"`
	ElectionKey   string   `json:"election_key"`
	NodeID        string   `json:"node_id"`      // default to hostname
	GracePeriod   int      `json:"grace_period"` // default to 10 seconds, unit is second

	// Writer node registry configurations
	WriterRegistryTTL int64 `json:"writer_registry_ttl"` // TTL for writer node registration in seconds, default 30
}

func NewNativePipelineTracer(cfg json.RawMessage) (*PipelineTracer, error) {
	pipelineTracer, err := tracer.NewPipelineTracer(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create pipeline tracer: %v", err)
	}
	t := &PipelineTracer{
		pipelineTracer: pipelineTracer,
	}
	return t, nil
}

func (t *PipelineTracer) OnBlockchainInit(chainConfig *params.ChainConfig) {
	t.pipelineTracer.OnBlockchainInit(chainConfig)
}

func (t *PipelineTracer) OnClose() {
	t.pipelineTracer.OnClose()
}

func (t *PipelineTracer) OnBlockStart(event tracing.BlockEvent) {
	t.pipelineTracer.OnBlockStart(event)
}

func (t *PipelineTracer) OnSystemCallStartHookV2(vm *tracing.VMContext) {
	t.pipelineTracer.OnSystemCallStartHookV2(vm)
}

func (t *PipelineTracer) OnBlockEnd(blockErr error) {
	t.pipelineTracer.OnBlockEnd(blockErr)
}

func (t *PipelineTracer) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.pipelineTracer.OnTxStart(vm, tx, from)
}

func (t *PipelineTracer) OnBorTxStart(txHash common.Hash) {
	//https://github.com/erigontech/erigon/blob/main/core/blockchain.go#L60
	tx := types.NewBorTransactionWithGasLimit(30_000_000)
	tx.SetHash(txHash)
	t.pipelineTracer.OnTxStart(nil, tx, common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe"))
}

func (t *PipelineTracer) OnTxEnd(receipt *types.Receipt, err error) {
	t.pipelineTracer.OnTxEnd(receipt, err)
}

func (t *PipelineTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.pipelineTracer.OnEnter(depth, typ, from, to, input, gas, value)
}

func (t *PipelineTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	t.pipelineTracer.OnExit(depth, output, gasUsed, err, reverted)
}

func (t *PipelineTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	t.pipelineTracer.OnOpcode(pc, op, gas, cost, scope, rData, depth, err)
}

func (t *PipelineTracer) OnLog(log *types.Log) {
	t.pipelineTracer.OnLog(log)
}

func (t *PipelineTracer) OnGenesisBlock(block *types.Block, alloc types.GenesisAlloc) {
	t.pipelineTracer.OnGenesisBlock(block, alloc)
}

func (t *PipelineTracer) OnBlockDBStart(db tracing.StateDB) {
	t.pipelineTracer.OnBlockDBStart(db)
}

func (t *PipelineTracer) OnCommit(originRoot common.Hash, root common.Hash, destructs map[common.Hash]struct{}, accounts map[common.Hash][]byte, accountsOrigin map[common.Address][]byte, storages map[common.Hash]map[common.Hash][]byte, storagesOrigin map[common.Address]map[common.Hash][]byte, codes map[common.Hash][]byte) {
	t.pipelineTracer.OnCommit(originRoot, root, destructs, accounts, accountsOrigin, storages, storagesOrigin, codes)
}

func (t *PipelineTracer) OnBalanceChange(addr common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	t.pipelineTracer.OnBalanceChange(addr, prev, new, reason)
}
