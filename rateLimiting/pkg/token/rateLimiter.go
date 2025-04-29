package token

import (
	"context"
	"errors"
	"log"
	"math"
	"sync"
	"time"
)

var (
	ErrUserNotFound     = errors.New("Пользователь с данным ID не найден")
	ErrUserAlreayExists = errors.New("Пользователь с таким ID уже существует")
)

type Response struct {
}

type RateLimiter struct {
	buckets map[string]*TokenBucket
	*sync.RWMutex
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		buckets: make(map[string]*TokenBucket),
		RWMutex: &sync.RWMutex{},
	}
}

func (rl *RateLimiter) GetOrCreateBucket(clientID string, capacity, refillRate float64) *TokenBucket {
	rl.Lock()
	defer rl.Unlock()

	if bucket, ok := rl.buckets[clientID]; ok {
		return bucket
	}

	log.Printf("New Client: IP: %s, Capacity: %f, RefillRate: %f", clientID, capacity, refillRate)

	bucket := NewTokenBucket(capacity, refillRate)
	rl.buckets[clientID] = bucket
	return bucket
}

func (rl *RateLimiter) AllowRequest(clientID string, capacity, refillRate float64) bool {
	bucket := rl.GetOrCreateBucket(clientID, capacity, refillRate)
	return bucket.Allow()
}

func (rl *RateLimiter) DeleteClient(clientID string) error {
	rl.Lock()
	defer rl.Unlock()

	if _, ok := rl.buckets[clientID]; ok {
		delete(rl.buckets, clientID)
		return nil
	}

	return ErrUserNotFound
}

func (rl *RateLimiter) AddClient(clientID string, capacity, rate float64) error {
	rl.RLock()
	if _, ok := rl.buckets[clientID]; ok {
		rl.RUnlock()
		return ErrUserAlreayExists
	}
	rl.RUnlock()

	rl.GetOrCreateBucket(clientID, capacity, rate)

	return nil
}

func (rl *RateLimiter) SetClientSettings(clientID string, capacity, rate float64) error {
	rl.RLock()
	client, ok := rl.buckets[clientID]
	rl.RUnlock()
	if !ok {
		return ErrUserNotFound
	}

	client.Lock()
	client.Capacity = capacity
	client.RefillRate = rate
	client.Tokens = math.Min(capacity, client.Tokens)
	client.Unlock()

	return nil
}

func (rl *RateLimiter) StartRefillTicker(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			rl.Lock()
			for _, bucket := range rl.buckets {
				bucket.Refill()
			}
			rl.Unlock()
		}
	}
}
