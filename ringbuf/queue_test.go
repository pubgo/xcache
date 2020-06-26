package ringbuf

import (
	"sync"
	"testing"
)

func TestQueue(t *testing.T) {
	q := &queue{Mutex: sync.Mutex{}}
	var g sync.WaitGroup
	for i := 0; i < 10; i++ {
		g.Add(1)
		go func(i int) {
			q.Push(uint32(i))
			g.Done()
		}(i)
	}
	g.Wait()
	//fmt.Printf("%#v,%d\n", q, q.Len())

	for i := 0; i < 10; i++ {
		g.Add(1)
		go func() {
			q.Pop()
			//fmt.Println(q.Pop())
			g.Done()
		}()
	}
	g.Wait()
	//fmt.Printf("%#v,%d\n", q, q.Len())
}
