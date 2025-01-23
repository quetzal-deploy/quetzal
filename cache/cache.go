package cache

import (
	"errors"
	"fmt"
	"github.com/rs/zerolog/log"
	"sync"
)

type StepData struct {
	Key   string
	Value string
}

type LockedMap[T any] struct {
	identifier string
	mutex      sync.RWMutex
	data       map[string]T
}

//type Cache struct {
//	mutex sync.RWMutex
//	data  map[string]string
//}

func NewLockedMap[T any](identifier string) LockedMap[T] {
	return LockedMap[T]{
		identifier: identifier,
		mutex:      sync.RWMutex{},
		data:       make(map[string]T),
	}
}

func (m LockedMap[T]) Update(key string, value T) {
	log.Info().
		Str("event", "write-"+m.identifier).
		Str("key", key).
		Any("value", value).
		Msg("write to " + m.identifier)

	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.data[key] = value
}

func (m LockedMap[T]) Get(key string) (T, error) {
	// FIXME: return error on m miss

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	data, ok := m.data[key]
	if ok {
		return data, nil
	} else {
		return data, errors.New(fmt.Sprintf("m: miss for '%s'", key))
	}
}

func (m LockedMap[T]) GetCopy() map[string]T {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	dataCopy := make(map[string]T)

	for key, value := range m.data {
		dataCopy[key] = value
	}

	return dataCopy
}
