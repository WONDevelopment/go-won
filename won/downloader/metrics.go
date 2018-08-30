// Copyright 2015 The go-ethereum Authors
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

// Contains the metrics collected by the downloader.

package downloader

import (
	"github.com/worldopennet/go-won/metrics"
)

var (
	headerInMeter      = metrics.NewRegisteredMeter("won/downloader/headers/in", nil)
	headerReqTimer     = metrics.NewRegisteredTimer("won/downloader/headers/req", nil)
	headerDropMeter    = metrics.NewRegisteredMeter("won/downloader/headers/drop", nil)
	headerTimeoutMeter = metrics.NewRegisteredMeter("won/downloader/headers/timeout", nil)

	bodyInMeter      = metrics.NewRegisteredMeter("won/downloader/bodies/in", nil)
	bodyReqTimer     = metrics.NewRegisteredTimer("won/downloader/bodies/req", nil)
	bodyDropMeter    = metrics.NewRegisteredMeter("won/downloader/bodies/drop", nil)
	bodyTimeoutMeter = metrics.NewRegisteredMeter("won/downloader/bodies/timeout", nil)

	receiptInMeter      = metrics.NewRegisteredMeter("won/downloader/receipts/in", nil)
	receiptReqTimer     = metrics.NewRegisteredTimer("won/downloader/receipts/req", nil)
	receiptDropMeter    = metrics.NewRegisteredMeter("won/downloader/receipts/drop", nil)
	receiptTimeoutMeter = metrics.NewRegisteredMeter("won/downloader/receipts/timeout", nil)

	stateInMeter   = metrics.NewRegisteredMeter("won/downloader/states/in", nil)
	stateDropMeter = metrics.NewRegisteredMeter("won/downloader/states/drop", nil)
)
