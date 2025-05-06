package main

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

var (
	ErrExecution = errors.New("execution error")
	ErrStopped   = errors.New("worker pool is stopped")
	ErrFull      = errors.New("worker pool is full")
)

type Option func(*WorkerPool)

func WithWorkersCount(workersCount int) Option {
	return func(pool *WorkerPool) {
		pool.workersCount = workersCount
	}
}

func WithQueueSize(poolSize int) Option {
	return func(pool *WorkerPool) {
		pool.queueSize = poolSize
	}
}

func WithRetryConfig(retriesCount int, delay time.Duration) Option {
	return func(pool *WorkerPool) {
		pool.retryConfig.MaxRetries = retriesCount
		pool.retryConfig.Delay = delay
	}
}

type Executor interface {
	Execute() (string, error)
}

type Result struct {
	Val string
	Err error
}

func NewWorkerPool(opts ...Option) *WorkerPool {
	wp := WorkerPool{
		workersCount: 1,
		queueSize:    1,
		stopChannel:  make(chan struct{}),
		retryConfig: RetryConfig{
			MaxRetries: 1,
			Delay:      0,
		},
	}

	for _, opt := range opts {
		opt(&wp)
	}

	wp.queue = make(chan ResultWithAttempts, wp.queueSize)
	wp.errQueue = make(chan ResultWithAttempts, wp.queueSize)
	wp.resultChan = make(chan Result, wp.queueSize)

	return &wp
}

type RetryConfig struct {
	MaxRetries int
	Delay      time.Duration
}

type ResultWithAttempts struct {
	Executor Executor
	Attempts int
}

type WorkerPool struct {
	workersCount int
	queueSize    int
	queue        chan ResultWithAttempts
	errQueue     chan ResultWithAttempts
	resultChan   chan Result
	stopChannel  chan struct{}
	wg           sync.WaitGroup
	retryConfig  RetryConfig
}

func (r *WorkerPool) Add(ctx context.Context, task Executor) error {
	select {
	case r.queue <- ResultWithAttempts{Executor: task, Attempts: 0}:
		return nil
	case <-r.stopChannel:
		return ErrStopped
	case <-ctx.Done():
		return ctx.Err()
	default:
		return ErrFull
	}
}

func (r *WorkerPool) Run(ctx context.Context) error {
	select {
	case <-r.stopChannel:
		return ErrStopped
	default:
	}

	for i := 0; i < r.workersCount; i++ {
		r.wg.Add(1)
		go r.process(ctx)
	}

	return nil
}

func (r *WorkerPool) process(ctx context.Context) {
	defer r.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopChannel:
			return
		case task, ok := <-r.queue:
			if !ok {
				return
			}

			val, err := task.Executor.Execute()
			if err != nil {
				if task.Attempts < r.retryConfig.MaxRetries {
					task.Attempts++
					r.errQueue <- task
					continue
				}
			}

			r.resultChan <- Result{Val: fmt.Sprintf("success attemps: %d, val: %s", task.Attempts, val), Err: nil}
		case task, ok := <-r.errQueue:
			if !ok {
				return
			}

			time.Sleep(r.retryConfig.Delay)

			val, err := task.Executor.Execute()
			if err != nil {
				if task.Attempts < r.retryConfig.MaxRetries {
					task.Attempts++
					r.errQueue <- task
					continue
				}

				r.resultChan <- Result{Val: "", Err: fmt.Errorf("err attemps: %d: %w: %s", task.Attempts, ErrExecution, err.Error())}
				continue
			}

			r.resultChan <- Result{Val: fmt.Sprintf("success attemps: %d, val: %s", task.Attempts, val), Err: nil}
		}

	}
}

func (r *WorkerPool) Stop() error {
	select {
	case <-r.stopChannel:
		return ErrStopped
	default:
	}

	close(r.stopChannel)
	r.wg.Wait()
	close(r.queue)
	close(r.errQueue)
	close(r.resultChan)

	return nil
}

func (r *WorkerPool) Results() <-chan Result {
	return r.resultChan
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	pool := NewWorkerPool(
		WithQueueSize(10),
		WithWorkersCount(1),
		WithRetryConfig(1, time.Millisecond*50),
	)

	err := pool.Run(ctx)
	if err != nil {
		fmt.Println(err)
	}

	go func() {
		for i := 0; i < 10; i++ {
			err := pool.Add(ctx, Task{})
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	go func() {
		time.Sleep(5 * time.Second)
		err = pool.Stop()
		if err != nil {
			fmt.Println(err)
		}
	}()

	for result := range pool.Results() {
		fmt.Println(result.Val, result.Err)
	}
}

type Task struct{}

func (r Task) Execute() (string, error) {
	time.Sleep(300 * time.Millisecond)

	taskNumb := rand.Intn(100)
	if rand.Intn(2)%2 > 0 {
		return fmt.Sprintf("result %d", taskNumb), nil
	} else {
		return "", fmt.Errorf("err %d: %w", taskNumb, ErrExecution)
	}
}
