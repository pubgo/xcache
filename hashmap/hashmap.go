package hashmap

import (
	"bytes"
	"github.com/pubgo/xcache/hashmap/internal"
	"math/rand"
	"time"
	"unsafe"
)

const defaultCap = 10

var entitySize = int(unsafe.Sizeof(entity{}))

type hashmap struct {
	cap         uint8
	entities    []*entity
	entities1   []*entity
	delEntities *entity

	slotsNum  uint32
	slotsNum1 uint32

	delNum uint32
	size   uint32
	count  uint32
	count1 uint32
}

type entity struct {
	key  uint8
	data []byte
	next *entity
}

func newHashmap() *hashmap {
	h := &hashmap{}
	h.cap = defaultCap
	h.slotsNum = 1<<h.cap - 1
	h.entities = make([]*entity, h.slotsNum+1)
	return h
}

func (h *hashmap) rehash(slot1 uint64) {
	if h.entities == nil {
		return
	}

	for h.entities1[slot1] != nil {
		ent := h.entities1[slot1]
		h.entities1[slot1] = ent.next
		h.count1--

		slot := internal.MemHash(ent.data[:ent.key]) & uint64(h.slotsNum)
		ent.next = h.entities[slot]
		h.entities[slot] = ent
		h.count++
	}
}

func (h *hashmap) rehash1() {
	if h.count1 > 0 {
		return
	}

	if h.entities1 != nil {
		h.entities1 = nil
	}

	if h.count > h.slotsNum*6 {
		h.cap++
	} else if h.count < h.slotsNum*2 && h.cap != defaultCap {
		h.cap--
	} else {
		return
	}

	h.slotsNum1 = h.slotsNum
	h.slotsNum = 1<<h.cap - 1

	h.entities1 = h.entities[:len(h.entities):len(h.entities)]
	h.entities = make([]*entity, h.slotsNum+1)

	h.count1 = h.count
	h.count = 0
}

func (h *hashmap) getSlots(key []byte) (uint64, uint64) {
	hk := internal.MemHash(key)
	return hk & uint64(h.slotsNum), hk & uint64(h.slotsNum1)
}

func (h *hashmap) get1(entities []*entity, slot uint64, key []byte) (ent, pre *entity) {
	for ent = entities[slot]; ent != nil; ent = ent.next {
		if bytes.Equal(ent.data[:ent.key], key) {
			return
		}
		pre = ent
	}
	return
}

func (h *hashmap) get(key []byte) *entity {
	var ent *entity
	slot, slot1 := h.getSlots(key)
	if h.entities1 != nil {
		ent, _ = h.get1(h.entities1, slot1, key)
		// 迁移数据
	}

	if ent == nil {
		ent, _ = h.get1(h.entities, slot, key)
	}
	return ent
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func (h *hashmap) del1(entities []*entity, slot uint64, key []byte) *entity {
	ent, pre := h.get1(entities, slot, key)
	if ent == nil {
		return ent
	}

	if pre == nil {
		entities[slot] = ent.next
	} else {
		pre.next = ent.next
	}

	ent.next = h.delEntities
	h.delEntities = ent
	h.delNum++
	h.size -= uint32(entitySize + len(ent.data))
	ent.data = ent.data[:0]
	h.rehash1()
	return ent
}

func (h *hashmap) del(key []byte) (ent *entity) {
	slot, slot1 := h.getSlots(key)
	if h.entities1 != nil {
		ent = h.del1(h.entities1, slot1, key)
		if ent != nil {
			h.count1--
		}
		h.rehash(slot1)
	}

	if ent == nil {
		ent = h.del1(h.entities, slot, key)
		if ent != nil {
			h.count--
		}
	}
	return
}

func (h *hashmap) set(key, val []byte) *entity {
	dl := len(key) + len(val)
	var dt = make([]byte, dl, dl)
	copy(dt[copy(dt, key):], val)

	var ent *entity
	slot, slot1 := h.getSlots(key)
	if h.entities1 != nil {
		ent, _ = h.get1(h.entities1, slot1, key)
		h.rehash(slot1)
	}

	if ent == nil {
		ent, _ = h.get1(h.entities, slot, key)
	}

	if ent == nil {
		if h.delEntities == nil {
			ent = &entity{key: uint8(len(key))}
		} else {
			ent = h.delEntities
			h.delEntities = h.delEntities.next
			h.delNum--
		}
		h.count++
	}

	//fmt.Println(h.count)
	//fmt.Println(len(ent.data), uint32(entitySize+dl-len(ent.data)))
	h.size += uint32(entitySize + dl - len(ent.data))
	ent.data = dt

	ent.next = h.entities[slot]
	h.entities[slot] = ent

	h.rehash1()
	return ent
}
