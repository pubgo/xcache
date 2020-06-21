package xcache

import (
	"time"
)

type MemClearStrategy uint8

const (
	d MemClearStrategy = 1 << iota
)

// ICache
type IXCache interface {
	Set(k, v []byte, e time.Duration) error
	Get(k []byte) ([]byte, error)
	GetSet(k, v []byte, e time.Duration) ([]byte, error)
	GetWithDataLoad(k []byte, e time.Duration, fn ...func(k []byte) (v []byte, err error)) ([]byte, error)
	GetExpiration(k []byte) (v []byte, expired int64, err error)
	Delete(k []byte) error
	DeleteExpired() error
	Init(opts ...Option) error
	Option() Options
}

// Options 缓存配置变量
type Options struct {
	DefaultExpiration time.Duration
	MinExpiration     time.Duration
	MaxExpiration     time.Duration

	MinBufSize   int
	MaxBufSize   uint32
	MaxBufFactor float32
	MaxBufExpand float32

	MinDataSize int
	MaxDataSize int

	DataLoadTime time.Duration
	ClearTime    time.Duration
	Delimiter    string

	// 缓存过期驱逐处理
	EvictedHandle func(key, value []byte)
	// 缓存过期清理策略
	// ExpiredHandle func(key, value []byte)
	// 缓存内存超限处理
	MemExceededHandle func(key, value []byte)

	// 防止雪崩策略
	SnowSlideStrategy func(expired time.Duration) time.Duration
	// 防止穿透策略
	PenetrateStrategy func(k, v []byte) (value []byte, e time.Duration)
	// 防止击穿策略
	BreakdownStrategy func()
}

// Option 可选配置
type Option func(o *Options)

func Init(opts ...Option) error {
	return defaultXCache.Init(opts...)
}

func DeleteExpired() error {
	return defaultXCache.DeleteExpired()
}

func Delete(k []byte) error {
	return defaultXCache.Delete(k)
}

func Set(k []byte, v []byte, e time.Duration) error {
	return defaultXCache.Set(k, v, e)
}

func Get(k []byte) ([]byte, error) {
	return defaultXCache.Get(k)
}

func GetSet(k []byte, v []byte, e time.Duration) ([]byte, error) {
	return defaultXCache.GetSet(k, v, e)
}

func GetWithExpiration(key []byte) (value []byte, tm int64, err error) {
	return defaultXCache.GetExpiration(key)
}

func SetDefaultCache(i IXCache) {
	defaultXCache = i
}

func GetOption() Options {
	return defaultXCache.Option()
}

func GetWithDataLoad(k []byte, e time.Duration, fn ...func(k []byte) (v []byte, err error)) ([]byte, error) {
	return defaultXCache.GetWithDataLoad(k, e, fn...)
}
