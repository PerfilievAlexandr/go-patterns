package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

var (
	ErrExecutionError    = errors.New("execution error")
	ErrWorkerPoolStopped = errors.New("worker pool is stopped")
)

type Execute interface {
	Execute() error
}

func NewWorkerPool(size int) *WorkerPool {
	pool := WorkerPool{
		workersCount: size,
		doneCh:       make(chan struct{}),
		tasks:        make([]Execute, 0),
	}
	pool.cond = sync.NewCond(&pool.mu)

	return &pool
}

type WorkerPool struct {
	workersCount int
	tasks        []Execute
	doneCh       chan struct{}
	wg           sync.WaitGroup
	mu           sync.Mutex
	cond         *sync.Cond
}

func (r *WorkerPool) Run(ctx context.Context) error {
	select {
	case <-r.doneCh:
		return ErrWorkerPoolStopped
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	for i := 0; i < r.workersCount; i++ {
		r.wg.Add(1)

		go func() {
			defer r.wg.Done()

			for {
				task, err := r.popTask(ctx)
				if err != nil {
					log.Println(err)
				}

				select {
				case <-r.doneCh:
					r.cond.Broadcast()
					return
				case <-ctx.Done():
					r.cond.Broadcast()
					return
				default:
					err = task.Execute()
					if err != nil {
						log.Println(ErrExecutionError)
					}
				}
			}

		}()
	}

	return nil
}

func (r *WorkerPool) Add(task Execute) error {
	select {
	case <-r.doneCh:
		return ErrWorkerPoolStopped
	default:
	}

	r.pushTask(task)

	return nil
}

func (r *WorkerPool) Stop() {
	select {
	case <-r.doneCh:
		return
	default:
		close(r.doneCh)
	}

	r.cond.Broadcast()
	r.cond.Broadcast()
	r.wg.Wait()
}

func (r *WorkerPool) pushTask(task Execute) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.tasks = append(r.tasks, task)
	r.cond.Signal()
}

func (r *WorkerPool) popTask(ctx context.Context) (Execute, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for len(r.tasks) == 0 {
		select {
		case <-r.doneCh:
			return nil, ErrWorkerPoolStopped
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
			r.cond.Wait()
		}
	}

	var currTask Execute
	currTask = r.tasks[0]
	r.tasks = r.tasks[1:]

	return currTask, nil
}

func main() {
	ctx, _ := context.WithCancel(context.Background())
	pool := NewWorkerPool(4)

	f1 := First{}
	f2 := First{}
	f3 := First{}
	f4 := First{}
	f5 := First{}

	err := pool.Run(ctx)
	if err != nil {
		fmt.Println(err)
	}

	pool.Add(&f1)
	pool.Add(&f2)
	pool.Add(&f3)
	pool.Add(&f4)
	pool.Add(&f5)

	time.Sleep(time.Second * 4)
	pool.Stop()
	pool.Stop()
	fmt.Println("Finish")
}

type First struct {
}

func (f *First) Execute() error {
	fmt.Println("First.Execute")
	time.Sleep(time.Second * 1)

	return nil
}
