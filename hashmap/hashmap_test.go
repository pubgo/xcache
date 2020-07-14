package hashmap

import (
	"fmt"
	"github.com/pubgo/xtest"
	"math/rand"
	"testing"
)

var h = newHashmap()
var m = make(map[string][]byte, 1024)

func TestName(t *testing.T) {
	var key = make([]byte, 100)
	bb := xtest.Benchmark(3000000).Do(func(b *xtest.B) {
		b.StopTimer()
		rand.Read(key)
		b.StartTimer()
		m[string(key)]=key
		//h.set(key, key)
		//ent := h.get(key)
		//if !bytes.Equal(ent.data[:ent.key], key) {
		//	t.Fatalf("%s %s", ent.data[:ent.key], key)
		//}
		//if ent != h.del(key) {
		//	t.Fatalf("%s %s", ent.data[:ent.key], key)
		//}
	})
	fmt.Println(h.size, h.count+h.count1, h.slotsNum, h.cap, h.delNum, bb)
	xtest.PrintMemStats()
}
