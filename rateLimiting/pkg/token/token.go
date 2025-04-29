package token

import (
	"math"
	"sync"
	"time"
)

type TokenBucket struct {
	Capacity   float64
	Tokens     float64
	RefillRate float64
	lastRefill time.Time
	*sync.RWMutex
}

func NewTokenBucket(capacity, refillRate float64) *TokenBucket {
	return &TokenBucket{
		Capacity:   capacity,
		Tokens:     capacity,
		RefillRate: refillRate,
		lastRefill: time.Now(),
		RWMutex:    &sync.RWMutex{},
	}
}

func (tb *TokenBucket) Refill() {
	tb.Lock()
	defer tb.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	tb.lastRefill = now

	tokensToAdd := elapsed.Seconds() * tb.RefillRate
	if tokensToAdd > 0 {
		tb.Tokens = math.Min(tb.Capacity, tb.Tokens+tokensToAdd)
	}
}

func (tb *TokenBucket) Allow() bool {
	tb.Lock()
	defer tb.Unlock()

	if tb.Tokens >= 1 {
		tb.Tokens -= 1
		return true
	}

	return false
}
