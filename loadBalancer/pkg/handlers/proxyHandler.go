package handlers

import (
	"log"
	"net/http"
	"net/http/httputil"
	"time"

	"loadBalancer/pkg/backend"
)

const (
	numRetries = 3
)

type CustomTransport struct {
	http.RoundTripper
	Retries int
	Pool    *backend.BackendPool
}

func (t *CustomTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	var err error
	var resp *http.Response
	var lastBackendURL string

	// Также можно было раширить структуру BackendPool каналом chan struct, который
	// посылает сигнал при выполнении PingServers и сервер при этом какой-то ожил,
	// тогда по сигналу из канала выполняется посылка запроса к живому серверу. При этом
	// можно запоминать в очереди запросы, которые приходили (очередь запросов)
	// В данном случае принято решение исплользовать политику retry-ев

	for i := 0; i <= t.Retries; i++ {
		b := t.Pool.NextBackend()
		if b == nil {
			log.Println("No available backends")
			return nil, err
		}

		currentBackendURL := b.URL.String()
		if lastBackendURL != "" && lastBackendURL != currentBackendURL {
			log.Printf("Switching from %s to %s", lastBackendURL, currentBackendURL)
		}
		lastBackendURL = currentBackendURL

		req.URL.Scheme = b.URL.Scheme
		req.URL.Host = b.URL.Host

		b.IncConn()
		defer b.DecConn()

		resp, err = t.RoundTripper.RoundTrip(req)
		if err == nil {
			b.SetAlive(true)
			return resp, nil
		}
		b.SetAlive(false)
		log.Printf("Retry %d: %v", i+1, err)
		time.Sleep(time.Second * time.Duration(i+1))
	}
	return resp, err
}

func SetupProxyHandler(pool *backend.BackendPool) http.Handler {
	proxy := &httputil.ReverseProxy{
		Rewrite: func(r *httputil.ProxyRequest) {
			b := pool.NextBackend()
			if b == nil {
				return
			}
			b.IncConn()
			defer b.DecConn()

			r.SetXForwarded()
			r.SetURL(b.URL)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Service unavailable", http.StatusServiceUnavailable)
		},
		Transport: &CustomTransport{
			RoundTripper: http.DefaultTransport,
			Retries:      numRetries,
			Pool:         pool,
		},
	}

	handler := &handler{proxy: proxy}

	return handler
}

type handler struct {
	proxy *httputil.ReverseProxy
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.proxy.ServeHTTP(w, r)
}
