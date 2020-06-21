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

// WithMaxBufSize ...
func WithMaxBufSize(maxBufSize uint32) Option {
	return func(o *Options) {
		o.MaxBufSize = maxBufSize
		o.MaxBufExpand = o.MaxBufFactor * float32(o.MaxBufSize)
	}
}

// WithMaxBufFactor ...
func WithMaxBufFactor(maxBufFactor float32) Option {
	return func(o *Options) {
		o.MaxBufFactor = maxBufFactor
		o.MaxBufExpand = o.MaxBufFactor * float32(o.MaxBufSize)
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
