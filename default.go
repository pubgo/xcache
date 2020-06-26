package xcache

import (
	"github.com/cespare/xxhash"
	"github.com/pubgo/xcache/consts"
	"github.com/pubgo/xcache/ringbuf"
	"github.com/pubgo/xcache/singleflight"
	"github.com/pubgo/xerror"
	"go.uber.org/atomic"
	"math/rand"
	"sync"
	"time"
)

var _ IXCache = (*xcache)(nil)
var emptyItem = item{}
var defaultXCache = func() *xcache {
	x, err := New()
	xerror.Exit(err)
	return x
}()

// New ...
func New(opts ...Option) (*xcache, error) {
	x := new(xcache)
	x.sg = new(singleflight.Group)
	x.rb = ringbuf.NewRingBuf()
	x.headItem = &headItem{
		dup:   make(map[string]item),
		items: make(map[uint32]item),
	}
	x = x.init()
	return x, x.Init(opts...)
}

type item struct {
	_        uint8
	key      uint8
	size     uint16
	index    uint32
	expireAt int64
}

type xcache struct {
	mutex    sync.Mutex
	opts     Options
	size     atomic.Uint32
	count    atomic.Uint32
	sg       *singleflight.Group
	rb       *ringbuf.RingBuf
	headItem *headItem
	janitor  *janitor
}

// GetWithDataLoad ...
func (x *xcache) GetWithDataLoad(k []byte, e time.Duration, fn ...func(k []byte) (v []byte, err error)) ([]byte, error) {
	return x.getSet(k, e, fn...)
}

// Options ...
func (x *xcache) Option() Options {
	return x.opts
}

func (x *xcache) init() *xcache {
	x.opts.DefaultExpiration = consts.DefaultExpiration
	x.opts.MinExpiration = consts.DefaultMinExpiration
	x.opts.MaxExpiration = consts.DefaultMaxExpiration
	x.opts.MinBufSize = consts.DefaultMinBufSize
	x.opts.MaxBufSize = consts.DefaultMaxBufSize
	x.opts.MinDataSize = consts.DefaultMinDataSize
	x.opts.MaxDataSize = consts.DefaultMaxDataSize
	x.opts.MaxKeySize = consts.DefaultMaxKeySize
	x.opts.DataLoadTime = consts.DefaultDataLoadTime
	x.opts.ClearTime = consts.DefaultClearTime
	x.opts.ClearRate = consts.DefaultClearNum
	x.opts.Delimiter = consts.DefaultDelimiter
	x.opts.SnowSlideStrategy = func(expired time.Duration) time.Duration {
		return expired + time.Duration(rand.Intn(int(x.opts.MinExpiration)))
	}
	x.opts.BreakdownStrategy = func(_ []byte, bytes []byte, dur time.Duration) ([]byte, time.Duration) {
		if bytes == nil || len(bytes) == 0 {
			return bytes, x.opts.SnowSlideStrategy(x.opts.MinExpiration)
		}
		return bytes, dur
	}
	x.opts.PenetrateStrategy = func(k []byte, fn ...func(k []byte) ([]byte, error)) ([]byte, error) {
		var ch = make(chan error, 1)
		var dt1 interface{}
		var err error
		dt1, err = x.sg.Do(string(k), func() (interface{}, error) {
			go func() {
				defer xerror.RespChanErr(ch)
				dt1, err = fn[0](k)
				ch <- xerror.WrapF(err, "key: %s", k)
			}()

			select {
			case <-time.After(x.opts.DataLoadTime):
				return nil, xerror.WrapF(ErrDataLoadTimeout, "key: %s", k)
			case err = <-ch:
				if err != nil {
					return nil, err
				}
			}
			return dt1, nil
		})
		if err != nil {
			return nil, err
		}

		return dt1.([]byte), nil
	}

	return x
}

// Init ...
func (x *xcache) Init(opts ...Option) error {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	opt := x.opts
	for _, o := range opts {
		o(&opt)
	}

	// 最大缓存判断
	if opt.MaxBufSize > consts.DefaultMaxBufSize || opt.MaxBufSize < consts.DefaultMinBufSize {
		return xerror.WrapF(ErrBufSize, "MaxBufSize: %d", opt.MaxBufSize)
	}

	// 过期时间判断
	{
		if opt.MaxExpiration > consts.DefaultMaxExpiration || opt.MaxExpiration < consts.DefaultMinExpiration {
			return xerror.WrapF(ErrExpiration, "MaxExpiration: %s", x.opts.MaxExpiration)
		}

		if opt.MinExpiration > consts.DefaultMaxExpiration || opt.MinExpiration < consts.DefaultMinExpiration {
			return xerror.WrapF(ErrExpiration, "MinExpiration: %s", opt.MinExpiration)
		}

		if opt.MinExpiration > x.opts.MaxExpiration {
			return xerror.WrapF(ErrExpiration, "MinExpiration: %s, MaxExpiration: %s", opt.MinExpiration, opt.MaxExpiration)
		}
	}

	// 默认过期时间判断
	if opt.DefaultExpiration > opt.MaxExpiration || opt.DefaultExpiration < consts.DefaultMinExpiration {
		return xerror.WrapF(ErrExpiration, "DefaultExpiration: %s", opt.DefaultExpiration)
	}

	// 数据长度判断
	{
		if opt.MaxDataSize > consts.DefaultMaxDataSize || opt.MaxDataSize < consts.DefaultMinDataSize {
			return xerror.WrapF(ErrLength, "MaxDataSize: %s", opt.MaxDataSize)
		}

		if opt.MinDataSize > consts.DefaultMaxDataSize || opt.MinDataSize < consts.DefaultMinDataSize {
			return xerror.WrapF(ErrLength, "MaxDataSize: %s", x.opts.MinDataSize)
		}

		if opt.MinDataSize > opt.MaxDataSize {
			return xerror.WrapF(ErrLength, "MinDataSize: %s, MaxDataSize: %s", opt.MinDataSize, opt.MaxDataSize)
		}

		if opt.MaxKeySize < consts.DefaultMinDataSize || opt.MaxKeySize > consts.DefaultMaxKeySize || opt.MaxKeySize > opt.MaxDataSize {
			return xerror.WrapF(ErrLength, "MaxKeySize: %s, MaxDataSizeL %s", opt.MaxKeySize, opt.MaxDataSize)
		}
	}

	// 过期清理时间校验
	if opt.ClearTime < consts.DefaultMinExpiration {
		return xerror.WrapF(ErrClearTime, "ClearTime: %s", opt.ClearTime)
	}

	// 数据加载时间校验
	if opt.DataLoadTime > consts.DefaultMaxExpiration || opt.DataLoadTime < consts.DefaultMinExpiration {
		return xerror.WrapF(ErrDataLoadTime, "DataLoadTime: %s", opt.ClearTime)
	}

	// 定期清理数据校验
	if opt.ClearRate < 0 {
		return xerror.WrapF(ErrClearNum, "clear_rate: %f", opt.ClearRate)
	}

	if err := x.initJanitor(); err != nil {
		return err
	}

	x.opts = opt
	return nil
}

func (x *xcache) checkKey(keySize int) error {
	if keySize > x.opts.MaxKeySize || keySize < x.opts.MinDataSize {
		return xerror.WrapF(ErrLength, "keySize: %d", keySize)
	}
	return nil
}

func (x *xcache) checkData(valSize int) error {
	if valSize > x.opts.MaxDataSize {
		return xerror.WrapF(ErrLength, "valSize: %d", valSize)
	}
	return nil
}

func (x *xcache) checkExpiration(expiration time.Duration) error {
	if expiration > x.opts.MaxExpiration || expiration < x.opts.MinExpiration {
		return xerror.WrapF(ErrExpiration, "expiration: %s", expiration)
	}
	return nil
}

func (x *xcache) getSet(k []byte, e time.Duration, fn ...func([]byte) ([]byte, error)) (dt []byte, err error) {
	defer xerror.RespErr(&err)

	xerror.Panic(x.checkKey(len(k)))

	h1 := x.hashKey(k)
	itm, _, existed := x.search(string(k), h1)
	if existed {
		if time.Now().UnixNano() < itm.expireAt {
			return x.rb.Get(itm.index)[itm.key:], nil
		}
		// 惰性过期清理
		go func() {
			_ = x.Delete(k)
		}()
	}

	// key不存在并且数据加载函数为nil
	if len(fn) == 0 || fn[0] == nil {
		return nil, ErrKeyNotFound
	}

	if x.opts.PenetrateStrategy != nil {
		dt, err = x.opts.PenetrateStrategy(k, fn...)
	} else {
		dt, err = fn[0](k)
	}

	if err != nil {
		return nil, err
	}

	dt, e = x.opts.BreakdownStrategy(k, dt, e)
	return dt, x.Set(k, dt, e)
}

// GetSet ...
func (x *xcache) GetSet(k []byte, v []byte, e time.Duration) (bt []byte, err error) {
	return x.getSet(k, e, func(bytes []byte) ([]byte, error) {
		return v, nil
	})
}

func (x *xcache) search(key string, h1 uint32) (item, keyType, bool) {
	return x.headItem.get(key, h1)
}

// Size ...
func (x *xcache) Size() uint32 {
	return x.size.Load()
}

// Len ...
func (x *xcache) Len() uint32 {
	return x.count.Load()
}

// SetDefault ...
func (x *xcache) SetDefault(key []byte, v []byte) error {
	return x.Set(key, v, x.opts.DefaultExpiration)
}

// Set ...
func (x *xcache) Set(key []byte, v []byte, e time.Duration) (err error) {
	defer xerror.RespErr(&err)

	keyLen := len(key)
	xerror.Panic(x.checkKey(keyLen))

	l := keyLen + len(v)
	xerror.Panic(x.checkData(l))

	xerror.Panic(x.checkExpiration(e))
	// 给时间设置随机性，防止雪崩
	if x.opts.SnowSlideStrategy != nil {
		e = x.opts.SnowSlideStrategy(e)
	}

	var dt = append(key, v...)

	// 内存超限处理
	{
		// 超过最大缓存, 直接报错
		bufSize := x.size.Add(uint32(l))
		if bufSize > x.opts.MaxBufSize {
			x.size.Sub(uint32(l))
			go func() {
				_ = x.DeleteExpired()
			}()
			return ErrBufExceeded
		}
	}

	var itm1 item
	itm1.key = uint8(keyLen)
	itm1.size = uint16(l)
	itm1.expireAt = time.Now().Add(e).UnixNano()

	h1 := x.hashKey(key)
	k := string(key)
	itm, kt, existed := x.search(k, h1)
	if existed {
		itm1.index = x.rb.Replace(itm.index, dt)
		x.headItem.set(k, h1, kt, itm1)
		x.size.Sub(uint32(l))
	} else {
		itm1.index = x.rb.Add(dt)
		x.headItem.set(k, h1, keyIndex, itm1)
		x.count.Inc()
	}
	return
}

func (x *xcache) hashKey(k []byte) uint32 {
	return uint32(xxhash.Sum64(k) >> 32)
}

// Get ...
func (x *xcache) Get(k []byte) ([]byte, error) {
	return x.getSet(k, x.opts.DefaultExpiration)
}

// Delete ...
func (x *xcache) Delete(key []byte) (err error) {
	xerror.RespErr(&err)

	xerror.Panic(x.checkKey(len(key)))

	h1 := x.hashKey(key)
	k := string(key)
	itm, kt, existed := x.search(k, h1)
	if !existed {
		return ErrKeyNotFound
	}

	x.headItem.del(k, h1, kt)
	x.size.Sub(uint32(itm.size))
	x.count.Dec()
	return nil
}

// 随机的找寻
func (x *xcache) randomDeleteExpired() {
	for _, itm := range x.headItem.randomExpired(x.opts.ClearRate) {
		x.headItem.del("", itm.h1, keyIndex)
		x.size.Sub(uint32(itm.size))
		x.count.Dec()
		x.sg.Clear()
	}
}

// DeleteExpired ...
func (x *xcache) DeleteExpired() error {
	x.headItem.dupClear()
	for _, itm := range x.headItem.randomExpired(1.0) {
		itm := itm
		x.headItem.del("", itm.h1, keyIndex)
		x.size.Sub(uint32(itm.size))
		x.count.Dec()
		x.sg.Clear()
	}
	return nil
}
