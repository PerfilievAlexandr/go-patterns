package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"
)

var (
	ErrNotAvailable = errors.New("service not available")
)

type Backend struct {
	Url   *url.URL
	Alive atomic.Bool
	Proxy *httputil.ReverseProxy
}

func (b *Backend) setAlive(isActive bool) {
	b.Alive.Store(isActive)
}

func (b *Backend) isAlive() bool {
	return b.Alive.Load()
}

func NewLoadBalancer(backends ...*Backend) *LoadBalancer {
	return &LoadBalancer{
		backends: backends,
		stopChan: make(chan struct{}),
	}
}

type LoadBalancer struct {
	backends []*Backend
	current  uint64
	stopChan chan struct{}
}

func (l *LoadBalancer) nextIndex() int {
	return int(atomic.AddUint64(&l.current, uint64(1)) % uint64(len(l.backends)))
}

func (l *LoadBalancer) nextBackend() (*Backend, error) {
	nextIdx := l.nextIndex()
	q := nextIdx + len(l.backends)

	for i := nextIdx; i < q; i++ {
		idx := i % len(l.backends)
		if l.backends[idx].isAlive() {
			if i != nextIdx {
				atomic.StoreUint64(&l.current, uint64(idx))
			}

			return l.backends[idx], nil
		}
	}

	return nil, ErrNotAvailable
}

func (l *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	back, err := l.nextBackend()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	back.Proxy.ServeHTTP(w, r)
}

func (l *LoadBalancer) HealthCheck(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	select {
	case <-ticker.C:
		for _, backend := range l.backends {
			go func(backend *Backend) {
				isAlive := isServiceAlive(backend.Url.Host)
				backend.setAlive(isAlive)
			}(backend)
		}
	case <-l.stopChan:
		return
	}
}

func (l *LoadBalancer) StopHealthCheck() {
	close(l.stopChan)
}

func isServiceAlive(address string) bool {
	conn, err := net.DialTimeout("tcp", address, time.Second)
	if err != nil {
		return false
	}
	_ = conn.Close()

	return true
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	serverMux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: serverMux,
	}

	var backends []*Backend
	addresses := []string{"http://127.0.0.1:8080", "http://127.0.0.1:8081"}
	for _, address := range addresses {
		urlParsed, err := url.Parse(address)
		if err != nil {
			log.Fatal(err)
		}

		proxy := httputil.NewSingleHostReverseProxy(urlParsed)
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("Proxy error: %v", err)
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

		backend := Backend{
			Url:   urlParsed,
			Proxy: proxy,
		}
		backend.Alive.Store(true)

		backends = append(backends, &backend)
	}

	loadBalancer := NewLoadBalancer(backends...)

	serverMux.Handle("/", loadBalancer)

	go loadBalancer.HealthCheck(time.Second * 5)
	defer loadBalancer.StopHealthCheck()

	if err := server.ListenAndServe(); err != nil {
		log.Fatal(err)
	}

	go func() {
		<-ctx.Done()
		contextWithTimout, cansel := context.WithTimeout(context.Background(), time.Second*5)
		defer cansel()

		err := server.Shutdown(contextWithTimout)
		if err != nil {
			log.Println("Server shutdown err:", err)
		}
	}()
}
