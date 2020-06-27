package ringbuf

import (
	"fmt"
	"strings"
	"testing"
)

var m = newRingBuf()

func BenchmarkGet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m.Get(uint32(i))
	}
}

func BenchmarkSet(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m.Add([]byte("hellohellohellohellohellohellohellohellohellohellohellohellohellohellohellohellohellohellohellohello"))
	}
}

func BenchmarkDel(b *testing.B) {
	for i := 0; i < b.N; i++ {
		m.Delete(uint32(i))
	}
}

func TestName(t *testing.T) {
	var a [][]byte
	for i := 1; i < 1000; i++ {
		a = append(a, []byte(strings.Repeat("h", i))[:i:i])
		fmt.Println(len(a), cap(a))
	}
	a = a[:len(a):len(a)]

	fmt.Println(len(a), cap(a))
	for _, v := range a {
		fmt.Println(len(v), cap(v))
	}
}
