package xcache

import (
	"sync"
)

type queue struct {
	sync.Mutex
	value []interface{}
}

// Len 长度判断
func (q *queue) Len() interface{} {
	q.Lock()
	defer q.Unlock()
	return len(q.value)
}

// Push ...
func (q *queue) Push(val interface{}) {
	q.Lock()
	defer q.Unlock()
	q.value = append(q.value, val)
}

// Pop ...
func (q *queue) Pop() interface{} {
	q.Lock()
	defer q.Unlock()

	if len(q.value) == 0 {
		return -1
	}

	val := q.value[len(q.value)-1]
	q.value = q.value[:len(q.value)-1]
	return val
}
