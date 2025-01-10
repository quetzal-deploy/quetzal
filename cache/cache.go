package cache

import "fmt"

type StepData struct {
	Key   string
	Value string
}

type Cache struct {
	data    map[string]string
	channel chan StepData
}

func NewCache() Cache {
	cache := Cache{
		data:    make(map[string]string),
		channel: make(chan StepData),
	}

	go cache.run()

	return cache
}

func (cache Cache) Update(msg StepData) error {
	fmt.Printf("cache: write to channel %v\n", cache.channel)
	cache.channel <- msg

	return nil
}

func (cache Cache) Get(key string) (string, error) {
	// FIXME: return error on cache miss

	return cache.data[key], nil
}

func (cache Cache) run() {
	for update := range cache.channel {
		fmt.Printf("cache update: %s = %s\n", update.Key, update.Value)
		cache.data[update.Key] = update.Value
	}
}
