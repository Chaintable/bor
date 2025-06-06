// Copyright 2024 The Erigon Authors
// This file is part of Erigon.
//
// Erigon is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// Erigon is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with Erigon. If not, see <http://www.gnu.org/licenses/>.

package tracer

import (
	"github.com/ethereum/go-ethereum/log"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
)

var systemAddress = common.HexToAddress("0xfffffffffffffffffffffffffffffffffffffffe")

func NewBorStateSyncTxnTracer(
	tracer *tracing.Hooks,
	stateSyncEventsCount int,
	stateReceiverContract common.Address,
) *tracing.Hooks {
	t := &borStateSyncTxnTracer{
		tracer:          tracer,
		remainingEvents: stateSyncEventsCount,
		totalEvents:     stateSyncEventsCount,
		stateReceiver:   stateReceiverContract,
		logger:          log.New("tracer", "borStateSyncTxnTracer"),
	}

	return &tracing.Hooks{
		OnTxStart:       t.OnTxStart,
		OnTxEnd:         t.OnTxEnd,
		OnEnter:         t.OnEnter,
		OnExit:          t.OnExit,
		OnOpcode:        t.OnOpcode,
		OnFault:         t.OnFault,
		OnGasChange:     t.OnGasChange,
		OnBalanceChange: t.OnBalanceChange,
		OnNonceChange:   t.OnNonceChange,
		OnCodeChange:    t.OnCodeChange,
		OnStorageChange: t.OnStorageChange,
		OnLog:           t.OnLog,
		OnCommit:        t.OnCommit,
		OnBorTxStart:    t.OnBorTxStart,
	}
}

type borStateSyncTxnTracer struct {
	tracer          *tracing.Hooks
	remainingEvents int
	totalEvents     int
	stateReceiver   common.Address
	reason          error
	logger          log.Logger
	depth           int
}

func (t *borStateSyncTxnTracer) OnTxStart(env *tracing.VMContext, tx *types.Transaction, from common.Address) {
	t.logger.Info("OnTxStart")
	if t.remainingEvents == t.totalEvents && t.tracer.OnTxStart != nil {
		// 触发虚拟交易开始
		t.tracer.OnTxStart(env, tx, from)
	}
}

func (t *borStateSyncTxnTracer) OnBorTxStart(env *tracing.VMContext, tx *types.Transaction, txHash common.Hash, from common.Address) {
	t.logger.Info("OnBorTxStart", "txHash", txHash.Hex(), "remainingEvents", t.remainingEvents, "totalEvents", t.totalEvents)
	if t.remainingEvents == t.totalEvents && t.tracer.OnBorTxStart != nil {
		t.tracer.OnBorTxStart(env, tx, txHash, from)
	}
}

func (t *borStateSyncTxnTracer) OnTxEnd(receipt *types.Receipt, err error) {
	t.logger.Info("OnTxEnd", "remainingEvents", t.remainingEvents, "totalEvents", t.totalEvents)
	if t.remainingEvents == 0 && t.tracer.OnTxEnd != nil {
		t.tracer.OnTxEnd(receipt, err)
	}
}

func (t *borStateSyncTxnTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	t.logger.Info("OnEnter", "depth", depth, "remainingEvents", t.remainingEvents)
	if t.tracer.OnEnter != nil {
		if t.depth == 0 {
			t.tracer.OnEnter(0, byte(vm.CALL), systemAddress, t.stateReceiver, nil, 0, big.NewInt(0))
		}
		t.depth++
		t.tracer.OnEnter(t.depth, typ, from, to, input, gas, value)
	}
}

func (t *borStateSyncTxnTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	t.logger.Info("OnExit", "depth", depth, "remainingEvents", t.remainingEvents, "totalEvents", t.totalEvents)
	if t.remainingEvents <= 0 {
		panic("unexpected extra exit event")
	}

	if t.tracer.OnExit != nil {
		t.tracer.OnExit(
			t.depth,
			output,
			gasUsed,
			err,
			reverted,
		)
		t.depth--
	}

	if depth == 0 {
		t.remainingEvents--
	}
	if t.remainingEvents == 0 {
		if t.tracer.OnExit != nil {
			t.tracer.OnExit(0, nil, 0, nil, false)
		}
	}
}

func (t *borStateSyncTxnTracer) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
	t.logger.Info("OnOpcode")
	if t.tracer.OnOpcode != nil {
		// trick tracer to think it is 1 level deeper
		t.tracer.OnOpcode(pc, op, gas, cost, scope, rData, t.depth, err)
	}
}

func (t *borStateSyncTxnTracer) OnFault(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, depth int, err error) {
	t.logger.Info("OnFault")
	if t.tracer.OnFault != nil {
		// trick tracer to think it is 1 level deeper
		t.tracer.OnFault(pc, op, gas, cost, scope, t.depth, err)
	}
}

// OnGasChange is called when gas is either consumed or refunded.
func (t *borStateSyncTxnTracer) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
	if t.tracer.OnGasChange != nil {
		t.tracer.OnGasChange(old, new, reason)
	}
}

func (t *borStateSyncTxnTracer) OnBlockStart(event tracing.BlockEvent) {
	if t.tracer.OnBlockStart != nil {
		t.tracer.OnBlockStart(event)
	}
}

func (t *borStateSyncTxnTracer) OnBlockEnd(err error) {
	if t.tracer.OnBlockEnd != nil {
		t.tracer.OnBlockEnd(err)
	}
}

func (t *borStateSyncTxnTracer) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
	if t.tracer.OnGenesisBlock != nil {
		t.tracer.OnGenesisBlock(b, alloc)
	}
}

func (t *borStateSyncTxnTracer) OnBalanceChange(a common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
	if t.tracer.OnBalanceChange != nil {
		t.tracer.OnBalanceChange(a, prev, new, reason)
	}
}

func (t *borStateSyncTxnTracer) OnNonceChange(a common.Address, prev, new uint64) {
	if t.tracer.OnNonceChange != nil {
		t.tracer.OnNonceChange(a, prev, new)
	}
}

func (t *borStateSyncTxnTracer) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
	if t.tracer.OnCodeChange != nil {
		t.tracer.OnCodeChange(a, prevCodeHash, prev, codeHash, code)
	}
}

func (t *borStateSyncTxnTracer) OnStorageChange(a common.Address, k common.Hash, prev, new common.Hash) {
	if t.tracer.OnStorageChange != nil {
		t.tracer.OnStorageChange(a, k, prev, new)
	}
}

func (t *borStateSyncTxnTracer) OnLog(log *types.Log) {
	t.logger.Info("OnLog", "log", log)
	if t.tracer.OnLog != nil {
		t.tracer.OnLog(log)
	}
}

func (t *borStateSyncTxnTracer) OnCommit(originRoot common.Hash, root common.Hash, destructs map[common.Hash]struct{}, accounts map[common.Hash][]byte, accountsOrigin map[common.Address][]byte, storages map[common.Hash]map[common.Hash][]byte, storagesOrigin map[common.Address]map[common.Hash][]byte, codes map[common.Hash][]byte) {
	if t.tracer.OnCommit != nil {
		t.tracer.OnCommit(originRoot, root, destructs, accounts, accountsOrigin, storages, storagesOrigin, codes)
	}
}
