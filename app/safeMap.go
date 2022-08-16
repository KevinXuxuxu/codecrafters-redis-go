package main

import (
	"fmt"
	"sync"
	"time"
)

type Value struct {
	data   string
	expire time.Time
}

type SafeMap struct {
	data map[string]Value
	lock sync.RWMutex
}

func newSafeMap() *SafeMap {
	return &SafeMap{map[string]Value{}, sync.RWMutex{}}
}

func (m *SafeMap) get(key string) (string, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()
	value, ok := m.data[key]
	if !ok || (!value.expire.IsZero() && value.expire.Before(time.Now())) {
		delete(m.data, key)
		return "", fmt.Errorf("key %s doesn't exist", key)
	}
	return value.data, nil
}

func (m *SafeMap) set(key string, value string) {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.data[key] = Value{value, time.Time{}}
}

func (m *SafeMap) setWithExpiry(key string, value string, expireAfterMs int) {
	m.lock.Lock()
	defer m.lock.Unlock()
	expire := time.Now().Add(time.Duration(expireAfterMs) * time.Millisecond)
	m.data[key] = Value{value, expire}
}
