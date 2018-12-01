// Copyright 2015 The go-bgmchain Authors
// This file is part of the go-bgmchain library.
//
// The go-bgmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-bgmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-bgmchain library. If not, see <http://www.gnu.org/licenses/>.

// Contains the metrics collected by the fetcher.

package fetcher

import (
	"github.com/5sWind/bgmchain/metrics"
)

var (
	propAnnounceInMeter   = metrics.NewMeter("bgm/fetcher/prop/announces/in")
	propAnnounceOutTimer  = metrics.NewTimer("bgm/fetcher/prop/announces/out")
	propAnnounceDropMeter = metrics.NewMeter("bgm/fetcher/prop/announces/drop")
	propAnnounceDOSMeter  = metrics.NewMeter("bgm/fetcher/prop/announces/dos")

	propBroadcastInMeter   = metrics.NewMeter("bgm/fetcher/prop/broadcasts/in")
	propBroadcastOutTimer  = metrics.NewTimer("bgm/fetcher/prop/broadcasts/out")
	propBroadcastDropMeter = metrics.NewMeter("bgm/fetcher/prop/broadcasts/drop")
	propBroadcastDOSMeter  = metrics.NewMeter("bgm/fetcher/prop/broadcasts/dos")

	headerFetchMeter = metrics.NewMeter("bgm/fetcher/fetch/headers")
	bodyFetchMeter   = metrics.NewMeter("bgm/fetcher/fetch/bodies")

	headerFilterInMeter  = metrics.NewMeter("bgm/fetcher/filter/headers/in")
	headerFilterOutMeter = metrics.NewMeter("bgm/fetcher/filter/headers/out")
	bodyFilterInMeter    = metrics.NewMeter("bgm/fetcher/filter/bodies/in")
	bodyFilterOutMeter   = metrics.NewMeter("bgm/fetcher/filter/bodies/out")
)
