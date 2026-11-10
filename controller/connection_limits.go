package controller

import "sync/atomic"

// Reserve before upgrade so concurrent handshakes cannot overshoot the limit.
func reserveConnection(counter *int64, limit int64) bool {
	for {
		current := atomic.LoadInt64(counter)
		if current >= limit {
			return false
		}
		if atomic.CompareAndSwapInt64(counter, current, current+1) {
			return true
		}
	}
}
