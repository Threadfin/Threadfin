package m3u

import (
	"sync"
)

// SafeList is a concurrent-safe list that holds interface{} items.
type SafeList struct {
	mu    sync.RWMutex
	items []interface{}
}

// Append adds an item to the list (write operation).
func (s *SafeList) Append(item interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items = append(s.items, item)
}

// Get returns an item from the list by index (read operation).
func (s *SafeList) Get(index int) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if index < 0 || index >= len(s.items) {
		return nil, false
	}
	return s.items[index], true
}

// Len returns the length of the list (read operation).
func (s *SafeList) Len() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.items)
}

// Contains checks if the list contains the specified item (read operation).
func (s *SafeList) Contains(item interface{}) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, v := range s.items {
		if v == item {
			return true
		}
	}
	return false
}
