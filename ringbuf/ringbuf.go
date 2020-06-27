package ringbuf

import (
	"math"
	"sync"
)

type ringBuf struct {
	data [][]byte
	q    queue
	sync.Mutex
}

func (r *ringBuf) ClearExpired() {
	r.Lock()
	defer r.Unlock()
	r.data = r.data[:len(r.data):len(r.data)]
}

func (r *ringBuf) Add(bytes []byte) uint32 {
	bytes = bytes[:len(bytes):len(bytes)]
	size := r.q.Pop()
	if size == math.MaxUint32 {
		r.Lock()
		defer r.Unlock()
		r.data = append(r.data, bytes)
		return uint32(len(r.data)) - 1
	}
	r.data[size] = bytes
	return size
}

func (r *ringBuf) Delete(u uint32) {
	r.q.Push(u)
}

func (r *ringBuf) Replace(u uint32, data []byte) uint32 {
	r.data[u] = data
	return u
}

func (r *ringBuf) Get(u uint32) []byte {
	return r.data[u]
}

func newRingBuf() *ringBuf {
	return &ringBuf{data: make([][]byte, 0)}
}

type RingBuf struct {
	*ringBuf
}

func NewRingBuf() *RingBuf {
	return &RingBuf{
		ringBuf: newRingBuf(),
	}
}
