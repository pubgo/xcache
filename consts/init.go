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
	DefaultMinBufSize = 10 << 20
	// 默认最大缓存1G, 超过会清理过期数据
	DefaultMaxBufSize = 1 << 30
	// 默认扩展缓存系数

	// 缓存数据最小长度, key
	DefaultMinDataSize = 5
	// 缓存数据最大长度
	DefaultMaxDataSize = 0xffff
	// Key最大长度
	DefaultMaxKeySize = 0xff

	// 默认分隔符
	DefaultDelimiter = "##"

	// 默认数据加载超时时间
	DefaultDataLoadTime = time.Second * 5

	// 默认过期定时清理时间1m
	DefaultClearTime = time.Minute * 5

	// 默认定期清理缓存数量为总数的10%
	DefaultClearNum = 0.1
)
