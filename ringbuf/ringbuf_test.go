package ringbuf

import "testing"

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
