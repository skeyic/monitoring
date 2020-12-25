package utils

import (
	"sync"
	"sync/atomic"
	"unsafe"
)

const mutexLocked = 1 << iota

// AsyncLocker ...
type AsyncLocker struct {
	sync.Mutex
}

// TryLock ...
func (l *AsyncLocker) TryLock() bool {
	return atomic.CompareAndSwapInt32((*int32)(unsafe.Pointer(&l.Mutex)), 0, mutexLocked)
}
