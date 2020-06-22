package xcache

import "sync"

type keyType uint8

const (
	keyDup keyType = iota + 1
	keyIndex
)

type headItem struct {
	mutex sync.RWMutex
	items map[uint16]map[uint16]item
	dup   map[string]item
}

func (x *headItem) get(key string, h1, h2 uint16) (item, keyType, bool) {
	x.mutex.RLock()
	defer x.mutex.RUnlock()

	keyHead, b := x.dup[key]
	if b {
		return keyHead, keyDup, true
	}

	if x.items[h1] == nil {
		return emptyItem, keyDup, false
	}

	keyHead = x.items[h1][h2]
	if keyHead.expireAt == 0 || keyHead.deleted {
		return emptyItem, keyIndex, false
	}

	return keyHead, keyIndex, true
}

func (x *headItem) set(key string, h1, h2 uint16, kt keyType, itm item) {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	if kt == keyIndex {
		if x.items[h1] == nil {
			x.items[h1] = make(map[uint16]item, keyCode)
		}
		x.items[h1][h2] = itm
	} else {
		x.dup[key] = itm
	}
}

func (x *headItem) del(key string, h1, h2 uint16, kt keyType) {
	x.mutex.Lock()
	defer x.mutex.Unlock()

	if kt == keyIndex {
		delete(x.items[h1], h2)
	} else {
		delete(x.dup, key)
	}
}
