package main

import (
	"fmt"
	"sync"
)

type SafeMap struct {
	data map[string]string
	lock sync.RWMutex
}

func newSafeMap() *SafeMap {
	return &SafeMap{map[string]string{}, sync.RWMutex{}}
}

func (m *SafeMap) get(key string) (string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	value, ok := m.data[key]
	if !ok {
		return "", fmt.Errorf("key %s doesn't exist", key)
	}
	return value, nil
}

func (m *SafeMap) set(key string, value string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.data[key] = value
}
