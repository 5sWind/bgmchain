// +build go1.4

package log

import "sync/atomic"

//
//
type swapHandler struct {
	handler atomic.Value
}
