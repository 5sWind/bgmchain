// Copyright 2016 The bgmchain Authors
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

package bgmclient

import "github.com/5sWind/bgmchain"

// Verify that Client implements the bgmchain interfaces.
var (
	_ = bgmchain.ChainReader(&Client{})
	_ = bgmchain.TransactionReader(&Client{})
	_ = bgmchain.ChainStateReader(&Client{})
	_ = bgmchain.ChainSyncReader(&Client{})
	_ = bgmchain.ContractCaller(&Client{})
	_ = bgmchain.GasEstimator(&Client{})
	_ = bgmchain.GasPricer(&Client{})
	_ = bgmchain.LogFilterer(&Client{})
	_ = bgmchain.PendingStateReader(&Client{})
	// _ = bgmchain.PendingStateEventer(&Client{})
	_ = bgmchain.PendingContractCaller(&Client{})
)
