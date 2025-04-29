package backend

import (
	"context"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type Backend struct {
	URL        *url.URL
	Alive      bool
	ActiveConn int64
	*sync.RWMutex
}

func (b *Backend) SetAlive(alive bool) {
	b.Lock()
	b.Alive = alive
	b.Unlock()
	log.Printf("Backend %s alive=%t", b.URL, alive)
}
func (b *Backend) IncConn()         { atomic.AddInt64(&b.ActiveConn, 1) }
func (b *Backend) DecConn()         { atomic.AddInt64(&b.ActiveConn, -1) }
func (b *Backend) ConnCount() int64 { return atomic.LoadInt64(&b.ActiveConn) }
func (b *Backend) IsAlive() bool {
	b.RLock()
	defer b.RUnlock()
	return b.Alive
}

type BackendPool struct {
	Backends []*Backend
	Strategy BalancerStrategy
	*sync.RWMutex
}

func NewBackendPool(urls []string) *BackendPool {
	const bufferChSize = 100

	pool := &BackendPool{RWMutex: &sync.RWMutex{}}
	for _, u := range urls {
		parsed, err := url.Parse(u)
		if err != nil {
			// Можно сделать, поскольку выполняется при инициализации приложения
			log.Fatalf("Неверный URL бэкэнда: %s", u)
		}
		b := &Backend{URL: parsed, Alive: true, RWMutex: &sync.RWMutex{}}
		pool.Backends = append(pool.Backends, b)
	}
	return pool
}

func (p *BackendPool) NextBackend() *Backend {
	p.RLock()
	if p.Strategy == nil {
		return nil
	}
	p.RUnlock()
	return p.Strategy.NextBackend(p)
}

func (p *BackendPool) getAliveBackends() []*Backend {
	var alive []*Backend

	p.RLock()
	backends := make([]*Backend, len(p.Backends))
	copy(backends, p.Backends)
	p.RUnlock()

	for _, b := range backends {
		if b.IsAlive() {
			alive = append(alive, b)
		}
	}

	return alive
}

func (p *BackendPool) SetStrategy(s BalancerStrategy) {
	p.Lock()
	p.Strategy = s
	p.Unlock()
}

// Функция проверки работоспособности сервера, если бы я реализовывал эти сервисы, то
// Реализовал бы в них ручку проверки состояние по типу /api/state/ или /api/health/,
// Вызывая которую сервис присылает ответ StatukOK, или же 500
func (p *BackendPool) HealthCheck(ctx context.Context, interval time.Duration) {
	wg := &sync.WaitGroup{}
	// При инициализации запускаем синхронно 1-ую проверку состояний серверов
	PingServers(p, wg)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("HealthCheck stopped: %v", ctx.Err())
			return
		case <-ticker.C:
			PingServers(p, wg)

		}
	}
}

func PingServers(pool *BackendPool, wg *sync.WaitGroup) {
	pool.RLock()
	backends := make([]*Backend, len(pool.Backends))
	copy(backends, pool.Backends)
	pool.RUnlock()

	wg.Add(len(backends))
	for _, b := range backends {
		go func(b *Backend) {
			defer wg.Done()
			// Пингуем сервер
			client := http.Client{Timeout: 2 * time.Second}
			_, err := client.Get(b.URL.String())
			alive := err == nil
			b.SetAlive(alive)
		}(b)
	}

	wg.Wait()
}

// Реализуем паттерн Стратегия, чтобы в runtime можно было подменять при необходимости алгоритм выбора сервера
type BalancerStrategy interface {
	NextBackend(pool *BackendPool) *Backend
}

type RoundRobinStrategy struct {
	counter uint64
}

func (r *RoundRobinStrategy) NextBackend(pool *BackendPool) *Backend {
	alive := pool.getAliveBackends()
	if len(alive) == 0 {
		return nil
	}
	idx := int(atomic.AddUint64(&r.counter, 1)) % len(alive)
	return alive[idx]
}

type RandomStrategy struct{}

func (r *RandomStrategy) NextBackend(pool *BackendPool) *Backend {
	alive := pool.getAliveBackends()
	if len(alive) == 0 {
		return nil
	}
	return alive[rand.Intn(len(alive))]
}

type LeastConnectionsStrategy struct{}

func (l *LeastConnectionsStrategy) NextBackend(pool *BackendPool) *Backend {
	alive := pool.getAliveBackends()
	if len(alive) == 0 {
		return nil
	}
	sort.Slice(alive, func(i, j int) bool {
		return alive[i].ConnCount() < alive[j].ConnCount()
	})

	return alive[0]
}
