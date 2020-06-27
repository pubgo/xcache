package xcache

import (
	"time"
)

type MemClearStrategy uint8

// ICache
type IXCache interface {
	Set(k, v []byte, e time.Duration) error
	Get(k []byte) ([]byte, error)
	GetSet(k, v []byte, e time.Duration) ([]byte, error)
	GetWithDataLoad(k []byte, e time.Duration, fn ...func(k []byte) (v []byte, err error)) ([]byte, error)
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

	MinBufSize int
	MaxBufSize uint32

	MinDataSize int
	MaxDataSize int
	MaxKeySize  int

	DataLoadTime time.Duration
	ClearTime    time.Duration
	ClearRate    float32
	Delimiter    string
	// 定期清理时间
	Interval time.Duration

	// 防止雪崩策略
	SnowSlideStrategy func(expired time.Duration) time.Duration
	// 防止穿透策略
	PenetrateStrategy func(k []byte, fn ...func(k []byte) ([]byte, error)) ([]byte, error)
	// 防止击穿策略
	BreakdownStrategy func([]byte, []byte, time.Duration) ([]byte, time.Duration)
}

// Option 可选配置
type Option func(o *Options)

func Init(opts ...Option) error {
	return defaultXCache.Init(opts...)
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

func GetOption() Options {
	return defaultXCache.Option()
}

func GetWithDataLoad(k []byte, e time.Duration, fn ...func(k []byte) (v []byte, err error)) ([]byte, error) {
	return defaultXCache.GetWithDataLoad(k, e, fn...)
}
