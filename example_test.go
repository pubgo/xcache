package xcache_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	fuzz "github.com/google/gofuzz"
	"github.com/pubgo/xcache"
	"testing"
	"time"
)

func BenchmarkCache(b *testing.B) {
	var fz = fuzz.New().NilChance(0).NumElements(10, 100)

	var key = []byte("njnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjnnjn")
	var c = xcache.New()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		fz.Fuzz(&key)
		b.StartTimer()
		if err := c.Set(key, key, time.Second); err != nil {
			panic(err)
		}
		val, err := c.Get(key)
		if err != nil {
			panic(err)
		}

		//fmt.Println(c.Size(), c.Len())

		if !bytes.Equal(key, val) {
			panic(hex.EncodeToString(key) + "\n\n" + hex.EncodeToString(val))
		}
	}
}
func TestName(t *testing.T) {
	var d interface{} = ([]byte)(nil)
	fmt.Println(d)
	fmt.Println(d.([]byte)==nil)
}
