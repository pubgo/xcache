package xcache

import (
	"github.com/cespare/xxhash"
	"github.com/pubgo/xcache/consts"
	"github.com/pubgo/xcache/singleflight"
	"github.com/pubgo/xerror"
	"go.uber.org/atomic"
	"math"
	"math/rand"
	"sync"
	"time"
)

const (
	key16   = 16
	keyCode = 1<<key16 - 1
)

var _ IXCache = (*xcache)(nil)
var emptyItem = item{}
var defaultXCache = func() IXCache {
	x := New()
	xerror.Exit(x.Init())
	return x
}()

// New ...
func New() *xcache {
	x := new(xcache)
	x.rb = newRingBuf()
	x.sg = new(singleflight.Group)
	x.headItem = &headItem{
		dup:   make(map[string]item),
		items: make(map[uint16]map[uint16]item, keyCode),
	}
	return x.init()
}

type item struct {
	keyLen   uint8
	deleted  bool
	size     uint16
	u        uint16
	u2       uint16
	expireAt int64
}

type xcache struct {
	mutex    sync.Mutex
	sg       *singleflight.Group
	opts     Options
	size     atomic.Uint32
	count    atomic.Uint32
	rb       *ringBuf
	headItem *headItem
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
	x.opts.MaxBufFactor = consts.DefaultMaxBufFactor
	x.opts.MaxBufExpand = consts.DefaultMaxBufExpand
	x.opts.MinDataSize = consts.DefaultMinDataSize
	x.opts.MaxDataSize = consts.DefaultMaxDataSize
	x.opts.DataLoadTime = consts.DefaultDataLoadTime
	x.opts.ClearTime = consts.DefaultClearTime
	x.opts.Delimiter = consts.DefaultDelimiter
	return x
}

// Init ...
func (x *xcache) Init(opts ...Option) error {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	for _, o := range opts {
		o(&x.opts)
	}

	// 最大缓存判断
	if x.opts.MaxBufSize > consts.DefaultMaxBufSize || x.opts.MaxBufSize < consts.DefaultMinBufSize {
		return xerror.WrapF(ErrBufSize, "MaxBufSize: %d", x.opts.MaxBufSize)
	}

	// 最大扩展缓存2G
	if x.opts.MaxBufExpand > math.MaxInt32 || x.opts.MaxBufExpand < consts.DefaultMinBufSize {
		return xerror.WrapF(ErrBufExceeded, "MaxBufExpand: %d", x.opts.MaxBufExpand)
	}

	// 过期时间判断
	{
		if x.opts.MaxExpiration > consts.DefaultMaxExpiration || x.opts.MaxExpiration < consts.DefaultMinExpiration {
			return xerror.WrapF(ErrExpiration, "MaxExpiration: %s", x.opts.MaxExpiration)
		}

		if x.opts.MinExpiration > consts.DefaultMaxExpiration || x.opts.MinExpiration < consts.DefaultMinExpiration {
			return xerror.WrapF(ErrExpiration, "MinExpiration: %s", x.opts.MinExpiration)
		}

		if x.opts.MinExpiration > x.opts.MaxExpiration {
			return xerror.WrapF(ErrExpiration, "MinExpiration: %s, MaxExpiration: %s", x.opts.MinExpiration, x.opts.MaxExpiration)
		}
	}

	// 默认过期时间判断
	if x.opts.DefaultExpiration > x.opts.MaxExpiration || x.opts.DefaultExpiration < consts.DefaultMinExpiration {
		return xerror.WrapF(ErrExpiration, "DefaultExpiration: %s", x.opts.DefaultExpiration)
	}

	// 数据长度判断
	{
		if x.opts.MaxDataSize > consts.DefaultMaxDataSize || x.opts.MaxDataSize < consts.DefaultMinDataSize {
			return xerror.WrapF(ErrLength, "MaxDataSize: %s", x.opts.MaxDataSize)
		}

		if x.opts.MinDataSize > consts.DefaultMaxDataSize || x.opts.MinDataSize < consts.DefaultMinDataSize {
			return xerror.WrapF(ErrLength, "MaxDataSize: %s", x.opts.MinDataSize)
		}

		if x.opts.MinDataSize > x.opts.MaxDataSize {
			return xerror.WrapF(ErrLength, "MinDataSize: %s, MaxDataSize: %s", x.opts.MinDataSize, x.opts.MaxDataSize)
		}
	}

	// 过期清理时间校验
	if x.opts.ClearTime > consts.DefaultMaxExpiration || x.opts.ClearTime < consts.DefaultMinExpiration {
		return xerror.WrapF(ErrClearTime, "ClearTime: %s", x.opts.ClearTime)
	}

	// 数据加载时间校验
	if x.opts.DataLoadTime > consts.DefaultMaxExpiration || x.opts.DataLoadTime < consts.DefaultMinExpiration {
		return xerror.WrapF(ErrDataLoadTime, "DataLoadTime: %s", x.opts.ClearTime)
	}

	return nil
}

// 过期时间处理
func (x *xcache) expiredHandle(d time.Duration) time.Duration {
	return d + time.Duration(rand.Intn(int(x.opts.MinExpiration)))
}

func (x *xcache) checkKey(keySize int) error {
	if keySize > x.opts.MaxDataSize || keySize < x.opts.MinDataSize {
		return xerror.WrapF(ErrLength, "keySize: %d", keySize)
	}
	return nil
}

func (x *xcache) checkValue(valSize int) error {
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

	h1, h2 := x.hashKey(k)
	itm, _, existed := x.search(string(k), h1, h2)
	if existed {
		if time.Now().UnixNano() < itm.expireAt {
			return x.rb.Get(itm.u, itm.u2)[itm.keyLen:], nil
		}
		// 过期立即删除
		go func() {
			_ = x.Delete(k)
		}()
	}

	// key不存在
	if len(fn) == 0 || fn[0] == nil {
		return nil, ErrKeyNotFound
	}

	var ch = make(chan error, 1)
	var dt1 interface{}
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

	dt = dt1.([]byte)
	var expired = x.opts.MinExpiration
	if dt != nil {
		expired = e
	}
	return dt, x.Set(k, dt, expired)
}

// GetSet ...
func (x *xcache) GetSet(k []byte, v []byte, e time.Duration) (bt []byte, err error) {
	return x.getSet(k, e, func(bytes []byte) ([]byte, error) {
		return v, nil
	})
}

func (x *xcache) search(key string, h1, h2 uint16) (item, keyType, bool) {
	return x.headItem.get(key, h1, h2)
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

	xerror.Panic(x.checkKey(len(key)))
	xerror.Panic(x.checkValue(len(v)))
	xerror.Panic(x.checkExpiration(e))

	var dt = make([]byte, len(key)+len(v))
	copy(dt[copy(dt, key):], v)

	{
		bufSize := x.Size() + uint32(len(dt))
		if float32(bufSize) > x.opts.MaxBufExpand {
			return xerror.WrapF(ErrBufExceeded, "bufSize: %d", bufSize)
		}

		if bufSize > x.opts.MaxBufSize {
			// 清空 singleflight
			go x.sg.Clear()
			go x.rb.ClearExpired()
		}
	}

	var itm1 item
	itm1.size = uint16(len(dt))
	itm1.keyLen = uint8(len(key))
	itm1.u = itm1.size >> 3
	itm1.expireAt = time.Now().Add(e).UnixNano()

	h1, h2 := x.hashKey(key)
	k := string(key)
	itm, kt, existed := x.search(k, h1, h2)
	if existed {
		itm1.u2 = x.rb.Replace(itm.u, itm.u2, dt)
		x.size.Sub(uint32(itm.size))
		x.headItem.set(k, h1, h2, kt, itm1)
	} else {
		itm1.u2 = x.rb.Add(dt)
		x.headItem.set(k, h1, h2, keyIndex, itm1)
		x.count.Add(1)
	}

	x.size.Add(uint32(itm1.size))
	return
}

func (x *xcache) hashKey(k []byte) (uint16, uint16) {
	keyHash := uint32(xxhash.Sum64(k) >> 32)
	return uint16(keyHash >> key16), uint16(keyHash & keyCode)
}

// Get ...
func (x *xcache) Get(k []byte) ([]byte, error) {
	return x.getSet(k, x.opts.DefaultExpiration, nil)
}

// GetExpiration ...
func (x *xcache) GetExpiration(key []byte) (value []byte, tm int64, err error) {
	xerror.RespErr(&err)

	xerror.Panic(x.checkKey(len(key)))

	hashKey1, hashKey2 := x.hashKey(key)
	itm, _, existed := x.search(string(key), hashKey1, hashKey2)
	if !existed {
		return nil, 0, ErrKeyNotFound
	}

	return x.rb.Get(itm.u, itm.u2)[itm.keyLen:], itm.expireAt, nil
}

// Delete ...
func (x *xcache) Delete(k []byte) (err error) {
	xerror.RespErr(&err)

	xerror.Panic(x.checkKey(len(k)))

	h1, h2 := x.hashKey(k)
	itm, kt, existed := x.search(string(k), h1, h2)
	if !existed {
		return xerror.WrapF(ErrKeyNotFound, "key: %s", k)
	}

	//if x.opts.EvictedHandle != nil {
	//	go func() {
	//		defer xerror.RespExit()
	//		x.opts.EvictedHandle(k, x.rb.Get(itm.u, itm.u2))
	//	}()
	//}

	x.headItem.del(string(k), h1, h2, kt)
	x.rb.Delete(itm.u, itm.u2)
	x.size.Sub(uint32(itm.size))
	x.count.Sub(1)
	return nil
}

// DeleteExpired ...
func (x *xcache) DeleteExpired() error {
	return nil
}

// Close ...
func (x *xcache) Close() error {
	x.rb.Close()
	*x = xcache{}
	return nil
}
