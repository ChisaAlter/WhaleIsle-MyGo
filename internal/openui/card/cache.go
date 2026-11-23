package card

import (
	"container/list"
	"sync"
)

// Cache keeps the cards of one surface by identity, so a card scrolled out
// of a virtualized transcript keeps its parsed program and state when it
// comes back. The least recently used card beyond the capacity is closed
// (its pending state is written first).
type Cache struct {
	mu    sync.Mutex
	cap   int
	order *list.List
	items map[string]*list.Element
}

type cacheEntry struct {
	key  string
	card *Card
}

// NewCache returns a cache holding up to capacity cards.
func NewCache(capacity int) *Cache {
	return &Cache{cap: max(1, capacity), order: list.New(), items: map[string]*list.Element{}}
}

// Get returns the card for key, creating it with create when absent.
func (c *Cache) Get(key string, create func() *Card) *Card {
	c.mu.Lock()
	if e, ok := c.items[key]; ok {
		c.order.MoveToFront(e)
		card := e.Value.(*cacheEntry).card
		c.mu.Unlock()
		return card
	}
	card := create()
	c.items[key] = c.order.PushFront(&cacheEntry{key, card})
	var evicted []*Card
	for c.order.Len() > c.cap {
		e := c.order.Back()
		c.order.Remove(e)
		ent := e.Value.(*cacheEntry)
		delete(c.items, ent.key)
		evicted = append(evicted, ent.card)
	}
	c.mu.Unlock()
	for _, old := range evicted {
		go old.Close()
	}
	return card
}

// Drop closes and forgets the card for key.
func (c *Cache) Drop(key string) {
	c.mu.Lock()
	e, ok := c.items[key]
	if ok {
		c.order.Remove(e)
		delete(c.items, key)
	}
	c.mu.Unlock()
	if ok {
		go e.Value.(*cacheEntry).card.Close()
	}
}

// Len is the number of cached cards.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.order.Len()
}
