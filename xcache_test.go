package xcache

import (
	"bytes"
	"fmt"
	"github.com/pubgo/xerror"
	"github.com/pubgo/xtest"
	"github.com/smartystreets/gunit"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	gunit.Run(new(xcacheFixture), t, gunit.Options.AllSequential())
}

type xcacheFixture struct {
	*gunit.Fixture
	unit *xcache
}

func (t *xcacheFixture) GetSet(k []byte, v []byte, e time.Duration) ([]byte, error) {
	panic("implement me")
}

func (t *xcacheFixture) Delete(k []byte) error {
	defer xerror.RespExit()

	err := t.unit.Delete(k)

	switch xerror.Unwrap(err) {
	case ErrLength:
		t.Assert(xerror.Is(t.unit.checkKey(len(k)), ErrLength))
	case ErrKeyNotFound:
		_, err = t.unit.Get(k)
		t.AssertEqual(err, ErrKeyNotFound)
	case nil:
	default:
		xerror.Exit(err)
	}

	return err
}

func (t *xcacheFixture) DeleteExpired() error {
	panic("implement me")
}

func (t *xcacheFixture) Close() error {
	panic("implement me")
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

func (t *xcacheFixture) TestSet() {
	fn := xtest.TestFuncWith(func(key, value []byte, e time.Duration) {
		defer xerror.RespExit()

		err := t.unit.Set(key, value, e)

		switch xerror.Unwrap(err) {
		case ErrLength:
			t.Assert(xerror.Is(t.unit.checkKey(len(key)), ErrLength) || xerror.Is(t.unit.checkData(len(value)+len(key)), ErrLength))
		case ErrExpiration:
			t.Assert(xerror.Is(t.unit.checkExpiration(e), ErrExpiration))
		case ErrBufExceeded:
			t.Assert(t.unit.Size()+uint32(len(append(key, value...))) >= t.unit.Option().MaxBufSize)
		case nil:
			val, err := t.unit.Get(key)
			t.AssertEqual(err, nil)
			t.Assert(bytes.Equal(val, value), fmt.Sprintf("%s, %s, %s, %s", key, val, value, err))
		default:
			xerror.Exit(err)
		}
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

func (t *xcacheFixture) TestGet() {
	fn := xtest.TestFuncWith(func(k []byte) {
		defer xerror.RespExit()

		_, r2 := t.unit.Get(k)
		switch xerror.Unwrap(r2) {
		case ErrLength:
			t.Assert(xerror.Is(t.unit.checkKey(len(k)), ErrLength))
		case ErrKeyNotFound:
			key := xtest.RangeBytes(10, 20)
			val := xtest.RangeBytes(10, 20)
			t.AssertEqual(t.unit.Set(key, val, xtest.RangeDur(time.Second*5, time.Minute)), nil)

			val1, err := t.unit.Get(key)
			t.AssertEqual(err, nil)
			t.AssertSprintEqual(val, val1)
		case nil:
		default:
			xerror.Exit(r2)
		}
	})
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
		
	}
}
