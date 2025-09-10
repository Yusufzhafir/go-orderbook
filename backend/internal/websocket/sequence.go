package websocket

import (
	"sync"
	"sync/atomic"
)

var seqMap sync.Map // map[string]*uint64

func nextSeq(symbol string) uint64 {
	v, _ := seqMap.LoadOrStore(symbol, new(uint64))
	ptr := v.(*uint64)
	return atomic.AddUint64(ptr, 1)
}
