package xcache

import (
	"github.com/cespare/xxhash"
	"github.com/pubgo/xcache/consts"
	"github.com/pubgo/xcache/singleflight"
	"github.com/pubgo/xerror"
	"math"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

const (
	key16   = 16
	keyCode = 1<<key16 - 1
)

type keyType uint8

const (
	keyTree keyType = iota + 1
	keyIndex
)

var emptyItem = item{}
var _ IXCache = (*xcache)(nil)
var defaultXCache = func() IXCache {
	x := New()
	xerror.Exit(x.Init())
	return x
}()

// New ...
func New() *xcache {
	x := new(xcache)
	x.dup = make(map[string]item)
	x.items = make(map[uint16]map[uint16]item, keyCode)
	x.rb = newRingBuf()
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
	mutex   sync.Mutex
	janitor *janitor
	sg      singleflight.Group
	opts    Options
	size    uint32
	count   uint32
	rb      *ringBuf
	items   map[uint16]map[uint16]item
	dup     map[string]item
	expired queue
	evicted func(k []byte, v []byte)
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

func (x *xcache) get(key string) (item, bool) {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	itm, b := x.dup[key]
	return itm, b
}

func (x *xcache) set(key string, itm item) {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	x.dup[key] = itm
}

func (x *xcache) del(key string) {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	delete(x.dup, key)
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

	hashKey1, hashKey2 := x.hashKey(k)
	itm, _, existed := x.search(k, hashKey1, hashKey2)
	if existed {
		if time.Now().UnixNano() < itm.expireAt {
			return x.rb.Get(itm.u, itm.u2)[itm.keyLen:], nil
		}
	}

	// key不存在
	if len(fn) == 0 {
		go x.expired.Push(k)
		return nil, ErrKeyNotFound
	}

	var ch = make(chan error, 1)
	dt1, err := x.sg.Do(string(k), func() (interface{}, error) {
		var dt1 interface{}
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

func (x *xcache) search(key []byte, h1, h2 uint16) (item, keyType, bool) {
	keyHead, b := x.get(string(key))
	if b {
		return keyHead, keyTree, true
	}

	if x.items[h1] == nil {
		x.items[h1] = make(map[uint16]item)
		return emptyItem, keyTree, false
	}

	keyHead = x.items[h1][h2]
	if keyHead.expireAt == 0 || keyHead.deleted {
		return emptyItem, keyIndex, false
	}

	return keyHead, keyIndex, true
}

// Size ...
func (x *xcache) Size() uint32 {
	return atomic.LoadUint32(&x.size)
}

// Len ...
func (x *xcache) Len() uint32 {
	return atomic.LoadUint32(&x.count)
}

// SetDefault ...
func (x *xcache) SetDefault(key []byte, v []byte) error {
	return x.Set(key, v, x.opts.DefaultExpiration)
}

// Set ...
func (x *xcache) Set(key []byte, v []byte, e time.Duration) (err error) {
	defer xerror.RespErr(&err)
	// 计算大小
	// 检查key value的长度和规范
	// 存储
	// 查询和替换

	// 检查
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
			xerror.Panic(x.DeleteExpired())
		}
	}

	var itm1 item
	itm1.size = uint16(len(dt))
	itm1.keyLen = uint8(len(key))
	itm1.u = itm1.size >> 3
	itm1.expireAt = time.Now().Add(e).UnixNano()

	hashKey1, hashKey2 := x.hashKey(key)
	itm, kt, existed := x.search(key, hashKey1, hashKey2)
	if existed {
		itm1.u2 = x.rb.Replace(itm.u, itm.u2, dt)
		atomic.AddUint32(&x.size, -uint32(itm.size))
		if kt == keyIndex {
			x.items[hashKey1][hashKey2] = itm1
		} else {
			x.dup[string(key)] = itm1
		}
	} else {
		itm1.u2 = x.rb.Add(dt)
		x.items[hashKey1][hashKey2] = itm1
		atomic.AddUint32(&x.count, 1)
	}

	atomic.AddUint32(&x.size, uint32(itm1.size))
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
	if err := x.checkKey(len(key)); err != nil {
		return nil, 0, err
	}

	hashKey1, hashKey2 := x.hashKey(key)
	itm, _, existed := x.search(key, hashKey1, hashKey2)
	if !existed {
		return nil, 0, ErrKeyNotFound
	}

	return x.rb.Get(itm.u, itm.u2)[itm.keyLen:], itm.expireAt, nil
}

// Delete ...
func (x *xcache) Delete(k []byte) error {
	hashKey1, hashKey2 := x.hashKey(k)
	itm, kt, existed := x.search(k, hashKey1, hashKey2)
	if !existed {
		return ErrKeyNotFound
	}

	if kt == keyIndex {
		x.items[hashKey1][hashKey2] = emptyItem
	} else {
		x.del(string(k))
	}

	if x.evicted != nil {
		x.evicted(k, x.rb.Get(itm.u, itm.u2))
	}

	x.rb.Delete(itm.u, itm.u2)
	atomic.AddUint32(&x.size, -uint32(itm.size))

	return nil
}

// DeleteExpired ...
func (x *xcache) DeleteExpired() error {
	//x.expired.Range(func(key, _ interface{}) bool {
	//	_ = x.Delete([]byte(key.(string)))
	//	return true
	//})
	return nil
}

// Close ...
func (x *xcache) Close() error {
	x.rb.Close()
	*x = xcache{}
	return nil
}
