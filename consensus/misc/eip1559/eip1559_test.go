// Copyright 2021 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package eip1559

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/stretchr/testify/require"
)

// copyConfig does a _shallow_ copy of a given config. Safe to set new values, but
// do not use e.g. SetInt() on the numbers. For testing only
func copyConfig(original *params.ChainConfig) *params.ChainConfig {
	return &params.ChainConfig{
		ChainID:                 original.ChainID,
		HomesteadBlock:          original.HomesteadBlock,
		DAOForkBlock:            original.DAOForkBlock,
		DAOForkSupport:          original.DAOForkSupport,
		EIP150Block:             original.EIP150Block,
		EIP155Block:             original.EIP155Block,
		EIP158Block:             original.EIP158Block,
		ByzantiumBlock:          original.ByzantiumBlock,
		ConstantinopleBlock:     original.ConstantinopleBlock,
		PetersburgBlock:         original.PetersburgBlock,
		IstanbulBlock:           original.IstanbulBlock,
		MuirGlacierBlock:        original.MuirGlacierBlock,
		BerlinBlock:             original.BerlinBlock,
		LondonBlock:             original.LondonBlock,
		TerminalTotalDifficulty: original.TerminalTotalDifficulty,
		Ethash:                  original.Ethash,
		Clique:                  original.Clique,
		Bor:                     original.Bor,
	}
}

func config() *params.ChainConfig {
	config := copyConfig(params.TestChainConfig)
	config.LondonBlock = big.NewInt(5)
	config.Bor.DelhiBlock = big.NewInt(8)

	return config
}

// TestBlockGasLimits tests the gasLimit checks for blocks both across
// the EIP-1559 boundary and post-1559 blocks
func TestBlockGasLimits(t *testing.T) {
	initial := new(big.Int).SetUint64(params.InitialBaseFee)

	for i, tc := range []struct {
		pGasLimit uint64
		pNum      int64
		gasLimit  uint64
		ok        bool
	}{
		// Transitions from non-london to london
		{10000000, 4, 20000000, true},  // No change
		{10000000, 4, 20019530, true},  // Upper limit
		{10000000, 4, 20019531, false}, // Upper +1
		{10000000, 4, 19980470, true},  // Lower limit
		{10000000, 4, 19980469, false}, // Lower limit -1
		// London to London
		{20000000, 5, 20000000, true},
		{20000000, 5, 20019530, true},  // Upper limit
		{20000000, 5, 20019531, false}, // Upper limit +1
		{20000000, 5, 19980470, true},  // Lower limit
		{20000000, 5, 19980469, false}, // Lower limit -1
		{40000000, 5, 40039061, true},  // Upper limit
		{40000000, 5, 40039062, false}, // Upper limit +1
		{40000000, 5, 39960939, true},  // lower limit
		{40000000, 5, 39960938, false}, // Lower limit -1
	} {
		parent := &types.Header{
			GasUsed:  tc.pGasLimit / 2,
			GasLimit: tc.pGasLimit,
			BaseFee:  initial,
			Number:   big.NewInt(tc.pNum),
		}
		header := &types.Header{
			GasUsed:  tc.gasLimit / 2,
			GasLimit: tc.gasLimit,
			BaseFee:  initial,
			Number:   big.NewInt(tc.pNum + 1),
		}
		err := VerifyEIP1559Header(config(), parent, header)
		if tc.ok && err != nil {
			t.Errorf("test %d: Expected valid header: %s", i, err)
		}

		if !tc.ok && err == nil {
			t.Errorf("test %d: Expected invalid header", i)
		}
	}
}

// TestCalcBaseFee assumes all blocks are 1559-blocks
func TestCalcBaseFee(t *testing.T) {
	t.Parallel()

	tests := []struct {
		parentBaseFee   int64
		parentGasLimit  uint64
		parentGasUsed   uint64
		expectedBaseFee int64
	}{
		{params.InitialBaseFee, 20000000, 10000000, params.InitialBaseFee}, // usage == target
		{params.InitialBaseFee, 20000000, 9000000, 987500000},              // usage below target
		{params.InitialBaseFee, 20000000, 11000000, 1012500000},            // usage above target
		{params.InitialBaseFee, 20000000, 20000000, 1125000000},            // usage full
		{params.InitialBaseFee, 20000000, 0, 875000000},                    // usage 0
	}
	for i, test := range tests {
		parent := &types.Header{
			Number:   big.NewInt(6),
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  big.NewInt(test.parentBaseFee),
		}
		if have, want := CalcBaseFee(config(), parent), big.NewInt(test.expectedBaseFee); have.Cmp(want) != 0 {
			t.Errorf("test %d: have %d  want %d, ", i, have, want)
		}
	}
}

// TestCalcBaseFeeDelhi assumes all blocks are 1559-blocks post Delhi Hard Fork
func TestCalcBaseFeeDelhi(t *testing.T) {
	t.Parallel()

	testConfig := copyConfig(config())

	// Test Delhi Hard Fork
	// Hard fork kicks in at block 8

	tests := []struct {
		parentBaseFee   int64
		parentGasLimit  uint64
		parentGasUsed   uint64
		expectedBaseFee int64
	}{
		{params.InitialBaseFee, 20000000, 10000000, params.InitialBaseFee}, // usage == target
		{params.InitialBaseFee, 20000000, 9000000, 993750000},              // usage below target
		{params.InitialBaseFee, 20000000, 11000000, 1006250000},            // usage above target
		{params.InitialBaseFee, 20000000, 20000000, 1062500000},            // usage full
		{params.InitialBaseFee, 20000000, 0, 937500000},                    // usage 0

	}
	for i, test := range tests {
		parent := &types.Header{
			Number:   big.NewInt(8),
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  big.NewInt(test.parentBaseFee),
		}
		if have, want := CalcBaseFee(testConfig, parent), big.NewInt(test.expectedBaseFee); have.Cmp(want) != 0 {
			t.Errorf("test %d: have %d  want %d, ", i, have, want)
		}
	}
}

// TestCalcBaseFeeNilParent tests that CalcBaseFee doesn't panic when
// the parent's BaseFee is nil.
func TestCalcBaseFeeNilParent(t *testing.T) {
	t.Parallel()

	testConfig := config()

	t.Run("nil baseFee for post-London parent returns InitialBaseFee", func(t *testing.T) {
		// Create a post-London parent header with nil BaseFee
		parent := &types.Header{
			Number:   big.NewInt(6), // Post-London because LondonBlock is 5 in test config
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  nil,
		}

		// CalcBaseFee should not panic but return InitialBaseFee
		result := CalcBaseFee(testConfig, parent)
		expected := big.NewInt(params.InitialBaseFee)

		require.NotNil(t, result, "CalcBaseFee should not return nil")
		require.Equal(t, expected, result,
			"CalcBaseFee should return InitialBaseFee when the parent's BaseFee is nil for post-London block")
	})

	t.Run("pre-London parent with nil baseFee returns InitialBaseFee", func(t *testing.T) {
		// Pre-London blocks should have nil BaseFee anyway
		parent := &types.Header{
			Number:   big.NewInt(4), // Pre-London because LondonBlock is 5 in test config
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  nil,
		}

		result := CalcBaseFee(testConfig, parent)
		expected := big.NewInt(params.InitialBaseFee)

		require.NotNil(t, result, "CalcBaseFee should not return nil")
		require.Equal(t, expected, result,
			"CalcBaseFee should return InitialBaseFee for first EIP-1559 block")
	})
}

// TestVerifyEIP1559HeaderNilParentBaseFee tests that VerifyEIP1559Header rejects post-London parents with nil BaseFee.
func TestVerifyEIP1559HeaderNilParentBaseFee(t *testing.T) {
	t.Parallel()

	testConfig := config()

	t.Run("post-London parent with nil BaseFee is rejected", func(t *testing.T) {
		// Malicious parent: post-London block with nil BaseFee
		parent := &types.Header{
			Number:   big.NewInt(6), // Post-London (LondonBlock is 5)
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  nil,
		}

		// Child header with valid BaseFee
		child := &types.Header{
			Number:   big.NewInt(7),
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}

		// VerifyEIP1559Header must reject due to nil parent's BaseFee
		err := VerifyEIP1559Header(testConfig, parent, child)
		require.Error(t, err, "VerifyEIP1559Header must reject nil parent's BaseFee")
		require.Contains(t, err.Error(), "parent header is missing baseFee",
			"Error message should indicate parent BaseFee is missing")
	})

	t.Run("pre-London parent with nil BaseFee is accepted", func(t *testing.T) {
		parent := &types.Header{
			Number:   big.NewInt(4), // Pre-London (LondonBlock is 5)
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  nil, // Expected for pre-London blocks
		}

		child := &types.Header{
			Number:   big.NewInt(5), // LondonBlock
			GasLimit: 40000000,      // parent.GasLimit * elasticityMultiplier = 20M * 2
			GasUsed:  20000000,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}

		err := VerifyEIP1559Header(testConfig, parent, child)
		require.NoError(t, err, "First London block with InitialBaseFee should be accepted")
	})

	t.Run("post-London parent with valid BaseFee is accepted", func(t *testing.T) {
		// Valid parent
		parent := &types.Header{
			Number:   big.NewInt(6), // Post-London (LondonBlock is 5)
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}

		// Valid child
		expectedBaseFee := CalcBaseFee(testConfig, parent)
		child := &types.Header{
			Number:   big.NewInt(7),
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  expectedBaseFee,
		}

		err := VerifyEIP1559Header(testConfig, parent, child)
		require.NoError(t, err, "Valid parent and child should be accepted")
	})
}

// TestBatchVerification tests that if a peer sends header batch [A, B] where A has nil BaseFee and future
// timestamp, and B is a child of A, it should not panic but return an error.
func TestBatchVerification(t *testing.T) {
	t.Parallel()

	testConfig := config()

	t.Run("batch A->B does not panic", func(t *testing.T) {
		// Header A: post-London, nil BaseFee, forwarded to child verification in batch
		headerA := &types.Header{
			Number:   big.NewInt(6), // Post-London (LondonBlock is 5)
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  nil,
		}

		// Header B: child of A, trying to exploit the nil BaseFee
		headerB := &types.Header{
			Number:   big.NewInt(7),
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}

		// verify B with A as parent doesn't panic but returns the expected error
		var err error
		require.NotPanics(t, func() {
			err = VerifyEIP1559Header(testConfig, headerA, headerB)
		}, "VerifyEIP1559Header must not panic when parent.BaseFee is nil")

		require.Error(t, err, "VerifyEIP1559Header must reject child when parent.BaseFee is nil")
		require.Contains(t, err.Error(), "parent header is missing baseFee",
			"Error must indicate parent BaseFee issue")
	})

	t.Run("CalcBaseFee called directly does not panic", func(t *testing.T) {
		// Header with nil BaseFee
		header := &types.Header{
			Number:   big.NewInt(6), // Post-London (LondonBlock is 5)
			GasLimit: 20000000,
			GasUsed:  10000000,
			BaseFee:  nil,
		}

		// CalcBaseFee doesn't panic
		var result *big.Int
		require.NotPanics(t, func() {
			result = CalcBaseFee(testConfig, header)
		}, "CalcBaseFee must not panic when parent.BaseFee is nil")

		require.NotNil(t, result, "CalcBaseFee should return non-nil result")
		require.Equal(t, big.NewInt(params.InitialBaseFee), result,
			"CalcBaseFee should return InitialBaseFee as fallback")
	})
}

func TestCalcParentGasTarget(t *testing.T) {
	t.Parallel()

	testConfig := copyConfig(config())
	testConfig.Bor.DandeliBlock = big.NewInt(20)

	defaultGasLimit := uint64(60_000_000)

	t.Run("gas target calculation pre dandeli HF", func(t *testing.T) {
		block := &types.Header{
			Number:   big.NewInt(9),
			GasLimit: defaultGasLimit,
			GasUsed:  defaultGasLimit / 2,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}
		gasTarget := calcParentGasTarget(testConfig, block)
		expected := block.GasLimit / 2 // because elasticity multiplier is set to 2 by default
		require.Equal(t, expected, gasTarget, "expected gas target = gaslimit/2")
	})

	t.Run("gas target calculation post dandeli HF", func(t *testing.T) {
		block := &types.Header{
			Number:   big.NewInt(20),
			GasLimit: defaultGasLimit,
			GasUsed:  defaultGasLimit / 2,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}
		gasTarget := calcParentGasTarget(testConfig, block)
		expected := block.GasLimit * params.TargetGasPercentagePostDandeli / 100 // because gas target is derived by this protocol parameter
		require.Equal(t, expected, gasTarget, "case #1: expected gas target = 60 percent of gas limit")

		block = &types.Header{
			Number:   big.NewInt(21),
			GasLimit: defaultGasLimit,
			GasUsed:  defaultGasLimit / 2,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}
		gasTarget = calcParentGasTarget(testConfig, block)
		expected = block.GasLimit * params.TargetGasPercentagePostDandeli / 100 // because gas target is derived by this protocol parameter
		require.Equal(t, expected, gasTarget, "case #2: expected gas target = 60 percent of gas limit")
	})

	t.Run("nil bor config", func(t *testing.T) {
		testConfig.Bor = nil
		block := &types.Header{
			Number:   big.NewInt(21),
			GasLimit: defaultGasLimit,
			GasUsed:  defaultGasLimit / 2,
			BaseFee:  big.NewInt(params.InitialBaseFee),
		}
		gasTarget := calcParentGasTarget(testConfig, block)
		expected := block.GasLimit / 2 // because elasticity multiplier is set to 2 by default
		require.Equal(t, expected, gasTarget, "expected gas target = gaslimit/2 when bor config is nil")
	})
}

// simpleBaseFeeCalculator contains an overly simplified logic of base fee calculations useful for generating
// expected values in test cases. It assumes all blocks are post-bhilai HF.
func simpleBaseFeeCalculator(initialBaseFee int64, gasLimit, gasUsed uint64, targetGasPercentage uint64) uint64 {
	initial := big.NewInt(initialBaseFee)
	val := big.NewInt(1)
	val.Mul(val, initial)

	// Assuming tests are running post bhilai
	bfd := int64(params.BaseFeeChangeDenominatorPostBhilai)

	// Define a target gas based on given percentage
	target := gasLimit * targetGasPercentage / 100
	if gasUsed == target {
		return initial.Uint64()
	}

	// follow the simple formula to get the new base fee:
	// base fee = initialBaseFee +/- (initialBaseFee * gasUsedDelta / gasTarget / baseFeeChangeDenominator)

	var delta int64
	if gasUsed > target {
		delta = int64(gasUsed - target)
	} else {
		delta = int64(target - gasUsed)
	}

	val.Mul(val, big.NewInt(delta))
	val.Div(val, big.NewInt(bfd))
	val.Div(val, big.NewInt(int64(target)))

	if gasUsed > target {
		return initial.Add(initial, val).Uint64()
	} else {
		return initial.Sub(initial, val).Uint64()
	}
}

func TestCalcBaseFeeDandeli(t *testing.T) {
	t.Parallel()

	testConfig := copyConfig(config())
	// Create a new Bor config to avoid modifying the shared one
	borCopy := *testConfig.Bor
	testConfig.Bor = &borCopy
	testConfig.Bor.BhilaiBlock = big.NewInt(8)
	testConfig.Bor.DandeliBlock = big.NewInt(20)

	// Case 1: Create pre-dandeli cases where HF is defined in future. Validate
	// base fee calculations before HF kicks in. Base fee should be calculated
	// based on default elasticity multiplier.
	tests := []struct {
		name            string
		parentBaseFee   int64
		parentGasLimit  uint64
		parentGasUsed   uint64
		expectedBaseFee uint64
	}{
		{"usage == target", params.InitialBaseFee, 60_000_000, 30_000_000, params.InitialBaseFee},
		{"usage below target #1", params.InitialBaseFee, 60_000_000, 20_000_000, 994791667},
		{"usage below target #2", params.InitialBaseFee, 60_000_000, 10_000_000, 989583334},
		{"usage above target #1", params.InitialBaseFee, 60_000_000, 40_000_000, 1005208333},
		{"usage above target #2", params.InitialBaseFee, 60_000_000, 50_000_000, 1010416666},
		{"usage full", params.InitialBaseFee, 60_000_000, 60_000_000, 1015625000},
		{"usage 0", params.InitialBaseFee, 60_000_000, 0, 984375000},
	}
	for _, test := range tests {
		block := &types.Header{
			Number:   big.NewInt(8),
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  big.NewInt(test.parentBaseFee),
		}
		baseFee := CalcBaseFee(testConfig, block).Uint64()
		expectedBaseFee := simpleBaseFeeCalculator(block.BaseFee.Int64(), block.GasLimit, block.GasUsed, params.DefaultTargetGasPercentage)
		require.Equal(
			t,
			expectedBaseFee,
			baseFee,
			fmt.Sprintf("pre-dandeli base fee mismatch with expected value, test: %s, got: %d, want: %d", test.name, baseFee, expectedBaseFee),
		)
		// Also check with manually calculated base fee
		require.Equal(
			t,
			test.expectedBaseFee,
			baseFee,
			fmt.Sprintf("pre-dandeli base fee mismatch with manually calculated value, test: %s, got: %d, want: %d", test.name, baseFee, expectedBaseFee),
		)
	}

	// Case 2: Create post-dandeli cases where HF has kicked in. Validate base fee changes
	// based on the newly introduced protocol param: TargetGasPrecentage. Target gas limit
	// should be calculated based on this percentage value out of total gas limit. Base
	// fee should be changed accordingly.
	tests = []struct {
		name            string
		parentBaseFee   int64
		parentGasLimit  uint64
		parentGasUsed   uint64
		expectedBaseFee uint64
	}{
		{"usage == target (65%)", params.InitialBaseFee, 60_000_000, 39_000_000, params.InitialBaseFee},
		{"usage below target #1", params.InitialBaseFee, 60_000_000, 30_000_000, 996394231},
		{"usage below target #2", params.InitialBaseFee, 60_000_000, 10_000_000, 988381411},
		{"usage above target #1", params.InitialBaseFee, 60_000_000, 40_000_000, 1000400641},
		{"usage above target #2", params.InitialBaseFee, 60_000_000, 50_000_000, 1004407051},
		{"usage full", params.InitialBaseFee, 60_000_000, 60_000_000, 1008413461},
		{"usage 0", params.InitialBaseFee, 60_000_000, 0, 984375000},
	}
	for _, test := range tests {
		// Post-dandeli block #1
		block := &types.Header{
			Number:   big.NewInt(20),
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  big.NewInt(test.parentBaseFee),
		}
		baseFee := CalcBaseFee(testConfig, block).Uint64()
		expectedBaseFee := simpleBaseFeeCalculator(block.BaseFee.Int64(), block.GasLimit, block.GasUsed, params.TargetGasPercentagePostDandeli)
		require.Equal(
			t,
			expectedBaseFee,
			baseFee,
			fmt.Sprintf("post-dandeli #1: base fee mismatch with expected value, test: %s, got: %d, want: %d", test.name, baseFee, expectedBaseFee),
		)
		// Also check with manually calculated base fee
		require.Equal(
			t,
			test.expectedBaseFee,
			baseFee,
			fmt.Sprintf("post-dandeli #1: base fee mismatch with manually calculated value, test: %s, got: %d, want: %d", test.name, baseFee, expectedBaseFee),
		)

		// Post-dandeli block #2
		block = &types.Header{
			Number:   big.NewInt(21),
			GasLimit: test.parentGasLimit,
			GasUsed:  test.parentGasUsed,
			BaseFee:  big.NewInt(test.parentBaseFee),
		}
		baseFee = CalcBaseFee(testConfig, block).Uint64()
		expectedBaseFee = simpleBaseFeeCalculator(block.BaseFee.Int64(), block.GasLimit, block.GasUsed, params.TargetGasPercentagePostDandeli)
		require.Equal(
			t,
			expectedBaseFee,
			baseFee,
			fmt.Sprintf("post-dandeli #2: base fee mismatch with expected value, test: %s, got: %d, want: %d", test.name, baseFee, expectedBaseFee),
		)
		// Also check with manually calculated base fee
		require.Equal(
			t,
			test.expectedBaseFee,
			baseFee,
			fmt.Sprintf("post-dandeli #2: base fee mismatch with manually calculated value, test: %s, got: %d, want: %d", test.name, baseFee, expectedBaseFee),
		)
	}
}
