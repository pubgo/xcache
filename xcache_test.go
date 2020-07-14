package xcache

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cespare/xxhash"
	"github.com/pubgo/xerror"
	"github.com/pubgo/xtest"
	"github.com/smartystreets/gunit"
	"os"
	"runtime"
	"testing"
	"time"
	"unsafe"
)

func TestNew(t *testing.T) {
	gunit.Run(new(xcacheFixture), t, gunit.Options.AllSequential())
}

type xcacheFixture struct {
	*gunit.Fixture
	unit *xcache
}

func (t *xcacheFixture) Delete(k []byte) error {
	defer xerror.Resp(func(err xerror.XErr) {
		switch err.Unwrap() {
		case ErrLength:
			t.Assert(xerror.Is(t.unit.checkKey(len(k)), ErrLength))
		case ErrKeyNotFound:
			_, err := t.unit.Get(k)
			t.AssertEqual(err, ErrKeyNotFound)
		default:
			xerror.Exit(err)
		}
	})

	xerror.Panic(t.unit.Delete(k))
	return nil
}

func (t *xcacheFixture) Options() Options {
	panic("implement me")
}

func (t *xcacheFixture) Setup() {
	t.unit = xerror.PanicErr(New()).(*xcache)
}

func (t *xcacheFixture) Teardown() {
	t.unit = nil
}

func (t *xcacheFixture) Set(key, value []byte, e time.Duration) error {
	defer xerror.Resp(func(err xerror.XErr) {
		switch xerror.Unwrap(err) {
		case ErrLength:
			t.Assert(xerror.Is(t.unit.checkKey(len(key)), ErrLength) || xerror.Is(t.unit.checkData(len(value)+len(key)), ErrLength))
		case ErrExpiration:
			t.Assert(xerror.Is(t.unit.checkExpiration(e), ErrExpiration))
		case ErrBufExceeded:
			t.Assert(t.unit.Size()+uint32(len(append(key, value...))) >= t.unit.Option().MaxBufSize)
		default:
			xerror.Exit(err)
		}
	})

	xerror.Panic(t.unit.Set(key, value, e))
	val, err := t.unit.Get(key)
	xerror.Exit(err)
	t.Assert(bytes.Equal(val, value), fmt.Sprintf("%s, %s, %s, %s", key, val, value, err))
	return nil
}

func (t *xcacheFixture) TestSet() {
	fn := xtest.TestFuncWith(func(key, value []byte, e time.Duration) {
		_ = t.Set(key, value, e)
		return
	})
	fn.In(
		nil,
		xtest.RangeBytes(0, 1),
		xtest.RangeBytes(10, 20),
		xtest.RangeBytes(512*1024, 513*1024),
	)
	fn.In(
		nil,
		xtest.RangeBytes(0, 1),
		xtest.RangeBytes(10, 20),
		xtest.RangeBytes(512*1024, 513*1024),
	)
	fn.In(
		xtest.RangeDur(-5*time.Second, 2*time.Second),
		xtest.RangeDur(2*time.Second, time.Minute),
		xtest.RangeDur(time.Minute, time.Minute*2),
	)
	fn.Do()
}

func (t *xcacheFixture) Get(k []byte) ([]byte, error) {
	defer xerror.Resp(func(err xerror.XErr) {
		switch err.Unwrap() {
		case ErrLength:
			t.Assert(xerror.Is(t.unit.checkKey(len(k)), ErrLength))
		case ErrKeyNotFound:
			key := xtest.RangeBytes(10, 20)
			val := xtest.RangeBytes(10, 20)
			t.AssertEqual(t.unit.Set(key, val, xtest.RangeDur(time.Second*5, time.Minute)), nil)

			val1, err := t.unit.Get(key)
			t.AssertEqual(err, nil)
			t.AssertSprintEqual(val, val1)
		default:
			xerror.Exit(err)
		}
	})

	r1, r2 := t.unit.Get(k)
	xerror.Panic(r2)
	return r1, r2
}

func (t *xcacheFixture) TestGet() {
	fn := xtest.TestFuncWith(t.Get)
	fn.In(
		nil,
		xtest.RangeBytes(0, 1),
		xtest.RangeBytes(10, 20),
		xtest.RangeBytes(512*1024, 513*1024),
	)
	fn.Do()
}

func BenchmarkName(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		key := xtest.RangeBytes(99, 100)
		b.StartTimer()
		Set(key, key, time.Second*10)
		Get(key)
		Delete(key)
	}
}

func TestName(t *testing.T) {
	//xtest.PrintMemStats()
	//b1 := xtest.Benchmark(3000000, func(b *xtest.B) {
	//	key := xtest.RangeBytes(99, 100)
	//	xerror.Panic(Set(key, key, time.Second*10))
	//	go func() {
	//		time.Sleep(xtest.RangeDur(time.Millisecond*10,time.Second*5))
	//		Delete(key)
	//	}()
	//})
	//xtest.PrintMemStats()
	//fmt.Println(b1.MemBytes, Size(), Count())

	//fmt.Printf("\n\n")
	//xtest.PrintMemStats()
	//var b1 bitset.BitSet
	//b2 := xtest.Benchmark(3000000, func(b *xtest.B) {
	//	key := xtest.RangeBytes(99, 100)
	//	b1.Set(uint(xxhash.Sum64(key))).None()
	//})
	type item struct {
		_        uint8
		key      uint8
		size     uint16
		index    uint32
		expireAt int64
	}

	xtest.PrintMemStats()
	var ss = make([]*item, 1<<30)
	_ = ss
	xtest.PrintMemStats()
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	dd, err := json.MarshalIndent(m, "", "\t")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(dd))
	select {}
}

type node struct {
	val  int
	next *node
}

type entity struct {
	deleted bool
	key     uint8
	_       [6]byte
	data    []byte
	expired int64
	next    *entity
}

func TestName1(t *testing.T) {
	fmt.Println(unsafe.Sizeof(entity{}))
}

func jmp(key uint64, buckets uint32) uint32 {
	var b, j uint64

	if buckets <= 0 {
		buckets = 1
	}

	for j < uint64(buckets) {
		b = j
		key = key*2862933555777941757 + 1
		j = uint64(float64(b+1) * (float64(int64(1)<<31) / float64((key>>33)+1)))
	}

	return uint32(b)
}

func slot(key []byte, slot uint32) uint32 {
	return uint32(xxhash.Sum64(key)>>32) & slot
}

var sss = uint32(1 << 30)

func BenchmarkNam2e(b *testing.B) {
	for i := 0; i < b.N; i++ {
		slot([]byte("hello000hello000hello000hello000hello000hello000hello000hello000"), sss)
	}
}

var map1 = map[string]interface{}{"hellohellohellohellohellohellohellohellohellohellohellohello": 1}
var map2 = map[uint32]interface{}{1111111111: 1}

func BenchmarkName1(b *testing.B) {

	b.Run("map1", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = map1["hellohellohellohellohellohellohellohellohellohellohellohello"]
		}
	})

	b.Run("map2", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = map2[1111111111]
		}
	})
}

func TestName123(t *testing.T) {
	fmt.Println(os.ExpandEnv("$HOME/data/"))
}
