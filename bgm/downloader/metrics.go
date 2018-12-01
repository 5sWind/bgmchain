// Copyright 2015 The bgmchain Authors
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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/5sWind/bgmchain/metrics"
)

var (
	headerInMeter      = metrics.NewMeter("bgm/downloader/headers/in")
	headerReqTimer     = metrics.NewTimer("bgm/downloader/headers/req")
	headerDropMeter    = metrics.NewMeter("bgm/downloader/headers/drop")
	headerTimeoutMeter = metrics.NewMeter("bgm/downloader/headers/timeout")

	bodyInMeter      = metrics.NewMeter("bgm/downloader/bodies/in")
	bodyReqTimer     = metrics.NewTimer("bgm/downloader/bodies/req")
	bodyDropMeter    = metrics.NewMeter("bgm/downloader/bodies/drop")
	bodyTimeoutMeter = metrics.NewMeter("bgm/downloader/bodies/timeout")

	receiptInMeter      = metrics.NewMeter("bgm/downloader/receipts/in")
	receiptReqTimer     = metrics.NewTimer("bgm/downloader/receipts/req")
	receiptDropMeter    = metrics.NewMeter("bgm/downloader/receipts/drop")
	receiptTimeoutMeter = metrics.NewMeter("bgm/downloader/receipts/timeout")

	stateInMeter   = metrics.NewMeter("bgm/downloader/states/in")
	stateDropMeter = metrics.NewMeter("bgm/downloader/states/drop")
)
