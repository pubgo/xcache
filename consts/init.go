package consts

import "time"

const (
	// 默认过期时间30s
	DefaultExpiration = time.Second * 30
	// 默认最小过期时间2s
	DefaultMinExpiration = time.Second * 2
	// 默认最大过期时间1m
	DefaultMaxExpiration = time.Minute

	// 默认最小缓存10M
	DefaultMinBufSize = 10 * 1 << 20
	// 默认最大缓存, 超过会清理过期数据
	DefaultMaxBufSize = 500 * 1 << 20
	// 默认扩展缓存系数
	DefaultMaxBufFactor = 0.2
	// 默认最大扩展缓存, 超过扩展缓存会返回错误信息
	DefaultMaxBufExpand = DefaultMaxBufSize * (1 + DefaultMaxBufFactor)

	// 缓存数据最小长度, key
	DefaultMinDataSize = 5
	// 缓存数据最大长度, key,value
	DefaultMaxDataSize = 0xffff

	// 默认分隔符
	DefaultDelimiter = "##"

	// 默认数据加载超时时间
	DefaultDataLoadTime = time.Second * 5

	// 默认过期定时清理时间1m
	DefaultClearTime = time.Minute
)
