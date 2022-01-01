package collections

import (
	"container/list"
)

type keyVal struct {
	key interface{}
	val interface{}
}

// OrderedMap tracks the insertion ordering of elements in a map.
// It is analogous to the collections.OrderedDict of Python.
// It is not thread-safe.
type OrderedMap struct {
	items map[interface{}]*list.Element
	order *list.List
}

// NewOrderedMap allocates an ordered map with the given initial capacity.
// The capacity will grow as needed.
func NewOrderedMap(cap int) OrderedMap {
	return OrderedMap{
		items: make(map[interface{}]*list.Element, cap),
		order: list.New(),
	}
}

// Len returns the number of elements in the map
func (m OrderedMap) Len() int {
	return len(m.items)
}

// Get returns the value for a given key in the map and an existence flag.
func (m OrderedMap) Get(key interface{}) (interface{}, bool) {
	elem := m.items[key]
	if elem == nil {
		return nil, false
	}
	return elem.Value.(keyVal).val, true
}

// Set sets the value for a given key in the map and returns true iff the key did not already exist.
// If the key already exists its value is updated, but its recency is not.
func (m OrderedMap) Set(key, val interface{}) bool {
	elem := m.items[key]
	if elem != nil {
		elem.Value = keyVal{key, val}
		return false
	}

	elem = m.order.PushFront(keyVal{key, val})
	m.items[key] = elem
	return true
}

// Delete deletes the given key from the map, returning the corresponding value and an existence flag.
func (m OrderedMap) Delete(key interface{}) (interface{}, bool) {
	elem := m.items[key]
	if elem == nil {
		return nil, false
	}
	delete(m.items, key)
	return m.order.Remove(elem).(keyVal).val, true
}

// RangeInc iterates over the map in increasing order of insertion recency.
func (m OrderedMap) RangeInc(cb func(k, v interface{}) bool) {
	elem := m.order.Back()
	for elem != nil {
		kv := elem.Value.(keyVal)
		elem = elem.Prev() // this needs to happen before cb(...), since cb(...) might delete `elem`
		if !cb(kv.key, kv.val) {
			break
		}
	}
}

// RangeDec iterates over the map in decreasing order of insertion recency.
func (m OrderedMap) RangeDec(cb func(k, v interface{}) bool) {
	elem := m.order.Front()
	for elem != nil {
		kv := elem.Value.(keyVal)
		elem = elem.Next() // this needs to happen before cb(...), since cb(...) might delete `elem`
		if !cb(kv.key, kv.val) {
			break
		}
	}
}
