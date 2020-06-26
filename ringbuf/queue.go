package ringbuf

import (
	"math"
	"sync"
)

type queue struct {
	sync.Mutex
	value []uint32
}

// Len 长度判断
func (q *queue) Len() int {
	return len(q.value)
}

// Push ...
func (q *queue) Push(val uint32) {
	q.Lock()
	defer q.Unlock()

	q.value = append(q.value, val)
}

// Pop ...
func (q *queue) Pop() uint32 {
	q.Lock()
	defer q.Unlock()

	l := len(q.value)

	if l == 0 {
		return math.MaxUint32
	}

	val := q.value[l-1]
	q.value = q.value[:l-1]
	return val
}
