// Copyright 2017 The bgmchain Authors
// This file is part of the bgmchain library.
//
// The bgmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The bgmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the bgmchain library. If not, see <http://www.gnu.org/licenses/>.

package bgmash

import (
	"math/big"
	"testing"

	"github.com/5sWind/bgmchain/core/types"
)

// Tests that bgmash works correctly in test mode.
func TestTestMode(t *testing.T) {
	head := &types.Header{Number: big.NewInt(1), Difficulty: big.NewInt(100)}

	bgmash := NewTester()
	block, err := bgmash.Seal(nil, types.NewBlockWithHeader(head), nil)
	if err != nil {
		t.Fatalf("failed to seal block: %v", err)
	}
	head.Nonce = types.EncodeNonce(block.Nonce())
	head.MixDigest = block.MixDigest()
	if err := bgmash.VerifySeal(nil, head); err != nil {
		t.Fatalf("unexpected verification error: %v", err)
	}
}
