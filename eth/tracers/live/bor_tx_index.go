package live

import "github.com/ethereum/go-ethereum/core/types"

func borTxIndexForBlock(block *types.Block) uint {
	if block == nil {
		return 0
	}
	txs := block.Transactions()
	if len(txs) > 0 && txs[len(txs)-1].Type() == types.StateSyncTxType {
		return uint(len(txs) - 1)
	}
	return uint(len(txs))
}
