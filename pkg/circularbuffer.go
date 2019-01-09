package failsafe

import "sync"

type CircularBoolArray struct {
	lock  sync.RWMutex
	ring  []bool
	idx   int
	count int
}

func NewCircularBoolArray(size int) *CircularBoolArray {
	return &CircularBoolArray{
		ring:  make([]bool, size),
		idx:   0,
		count: 0,
	}
}

func (c *CircularBoolArray) Insert(val bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	idx := (c.idx + 1) % len(c.ring)
	if c.ring[idx] {
		c.count--
	}
	c.ring[idx] = val
	if val {
		c.count++
	}
	c.idx++
}

func (c *CircularBoolArray) Count() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.count
}

func (c *CircularBoolArray) CountFalse() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.ring) - c.count
}

func (c *CircularBoolArray) Reset(val bool) {
	c.lock.Lock()
	defer c.lock.Unlock()
	for i := 0; i < len(c.ring); i++ {
		c.ring[i] = val
	}
	if val {
		c.count = len(c.ring)
	} else {
		c.count = 0
	}
}
