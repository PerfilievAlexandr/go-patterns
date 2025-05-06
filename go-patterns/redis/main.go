package main

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"
)

var (
	ErrNoTTl   = errors.New("invalid ttl")
	ErrNoFound = errors.New("not found")
)

type Redis interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string, ttl time.Duration) error
	Del(ctx context.Context, key string) error
}

func WithCleanDuration(period time.Duration) Option {
	return func(storage *redis) {
		storage.cleanDuration = period
	}
}

type Option func(r *redis)

func NewRedis(ctx context.Context, options ...Option) Redis {
	storage := redis{
		data:          make(map[string]string),
		ttl:           make(map[string]time.Time),
		cleanDuration: time.Second * 10,
	}

	for _, opt := range options {
		opt(&storage)
	}

	go storage.runClean(ctx)

	return &storage
}

type redis struct {
	data          map[string]string
	ttl           map[string]time.Time
	mu            sync.RWMutex
	cleanDuration time.Duration
}

func (r *redis) Set(_ context.Context, key, value string, ttl time.Duration) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.data[key] = value

	if ttl > 0 {
		r.ttl[key] = time.Now().Add(ttl).UTC()
	} else if ttl == 0 {
		delete(r.ttl, key)
	} else {
		return ErrNoTTl
	}

	return nil
}

func (r *redis) Get(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ttl, ok := r.ttl[key]
	if ok && time.Now().After(ttl) {
		delete(r.ttl, key)
		delete(r.data, key)

		return "", ErrNoFound
	}

	val, ok := r.data[key]
	if !ok {
		return "", ErrNoFound
	}

	return val, nil
}

func (r *redis) Del(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.data, key)
	delete(r.ttl, key)

	return nil
}

func (r *redis) runClean(ctx context.Context) {
	ticker := time.NewTicker(r.cleanDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println(ctx.Err())
		case <-ticker.C:
			r.mu.Lock()
			for key, ttl := range r.ttl {
				if time.Now().UTC().After(ttl) {
					delete(r.data, key)
					delete(r.ttl, key)
				}
			}
			r.mu.Unlock()
		}
	}
}

func main() {

}
