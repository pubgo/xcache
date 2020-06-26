package xcache

import (
	"time"
)

// WithDefaultExpiration ...
func WithDefaultExpiration(defaultExpiration time.Duration) Option {
	return func(o *Options) {
		o.DefaultExpiration = defaultExpiration
	}
}

// WithMinExpiration ...
func WithMinExpiration(minExpiration time.Duration) Option {
	return func(o *Options) {
		o.MinExpiration = minExpiration
	}
}

// WithMaxExpiration ...
func WithMaxExpiration(maxExpiration time.Duration) Option {
	return func(o *Options) {
		o.MaxExpiration = maxExpiration
	}
}

// WithMinBufSize ...
func WithMinBufSize(minBufSize int) Option {
	return func(o *Options) {
		o.MinBufSize = minBufSize
	}
}

// WithMinDataSize ...
func WithMinDataSize(minDataSize int) Option {
	return func(o *Options) {
		o.MinDataSize = minDataSize
	}
}

// WithMaxDataSize ...
func WithMaxDataSize(maxDataSize int) Option {
	return func(o *Options) {
		o.MaxDataSize = maxDataSize
	}
}

// WithDataLoadTime ...
func WithDataLoadTime(mataLoadTime time.Duration) Option {
	return func(o *Options) {
		o.DataLoadTime = mataLoadTime
	}
}

// WithClearTime ...
func WithClearTime(clearTime time.Duration) Option {
	return func(o *Options) {
		o.ClearTime = clearTime
	}
}

func WithClearNum(clearRate float32) Option {
	return func(o *Options) {
		o.ClearRate = clearRate
	}
}

func WithSnowSlideStrategy(snowSlideStrategy func(expired time.Duration) time.Duration) Option {
	return func(o *Options) {
		o.SnowSlideStrategy = snowSlideStrategy
	}
}

func WithPenetrateStrategy(penetrateStrategy func(k []byte, fn ...func(k []byte) ([]byte, error)) ([]byte, error)) Option {
	return func(o *Options) {
		o.PenetrateStrategy = penetrateStrategy
	}
}

func WithBreakdownStrategy(breakdownStrategy func([]byte, []byte, time.Duration) ([]byte, time.Duration)) Option {
	return func(o *Options) {
		o.BreakdownStrategy = breakdownStrategy
	}
}
