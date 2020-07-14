package xcache

import (
	"time"
)

type keyType uint8

const (
	keyDup keyType = iota + 1
	keyIndex
)

type headItem struct {
	items map[uint32]item
	dup   map[string]item
}

type expiredItem struct {
	size  uint16
	index uint32
	h1    uint32
}

func (x *headItem) dupClear() {
	var dup = make(map[string]item, len(x.dup))

	for k, v := range x.dup {
		dup[k] = v
	}

	x.dup = dup
}

// 获取随机item的过期item
func (x *headItem) randomExpired(rate float32) []expiredItem {
	var n = int(rate * float32(len(x.items)))
	var items = make([]expiredItem, n)
	var now = time.Now().UnixNano()

	for h1, v1 := range x.items {
		if n == 0 {
			break
		}

		if v1.expireAt < now {
			items = append(items, expiredItem{h1: h1, index: v1.index, size: v1.size})
		}
		n--
	}

	return items
}

func (x *headItem) get(key string, h1 uint32) (item, keyType, bool) {
	keyHead, ok := x.dup[key]
	if ok {
		if keyHead.expireAt != 0 {
			return keyHead, keyDup, true
		}
		return emptyItem, keyDup, false
	}

	keyHead, ok = x.items[h1]
	if ok {
		if keyHead.expireAt != 0 {
			return keyHead, keyIndex, true
		}
		return emptyItem, keyIndex, false
	}

	return emptyItem, keyDup, false
}

func (x *headItem) set(key string, h1 uint32, kt keyType, itm item) {
	if kt == keyIndex {
		x.items[h1] = itm
	} else {
		x.dup[key] = itm
	}
}

func (x *headItem) del(key string, h1 uint32, kt keyType) {
	if kt == keyIndex {
		delete(x.items, h1)
	} else {
		delete(x.dup, key)
	}
}
