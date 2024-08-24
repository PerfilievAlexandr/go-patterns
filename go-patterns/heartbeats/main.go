package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	//ctx, cansel := context.WithCancel(context.Background())
	//time.AfterFunc(10*time.Second, func() {
	//	cansel()
	//})
	//
	//const timeout = 2 * time.Second
	//heartbeats, results := doWork(ctx, timeout/2)
	//
	//for {
	//	select {
	//	case _, ok := <-heartbeats:
	//		if !ok {
	//			return
	//		}
	//		fmt.Println("pulse")
	//	case r, ok := <-results:
	//		if !ok {
	//			return
	//		}
	//		fmt.Printf("results %v\n", r)
	//	case <-time.After(timeout):
	//		fmt.Println("worker goroutine is not healthy!")
	//		return
	//	}
	//}

	ctx := context.Background()

	heartbeats, results := doWorkWithHeartbeatBefore(ctx, 1, 4, 5, 6, 3, 45)

	for {
		select {
		case _, ok := <-heartbeats:
			if !ok {
				return
			}
			fmt.Println("pulse")
		case r, ok := <-results:
			if !ok {
				return
			}
			fmt.Printf("results %v\n", r)
		}
	}
}

func doWork(ctx context.Context, pulseInterval time.Duration) (<-chan int, <-chan time.Time) {
	heartbeats := make(chan int)
	results := make(chan time.Time)

	go func() {
		defer close(heartbeats)
		defer close(results)

		pulse := time.Tick(pulseInterval)
		workGen := time.Tick(2 * pulseInterval)

		sendPulse := func() {
			select {
			case heartbeats <- 1:
			default:
			}
		}
		sendResult := func(r time.Time) {
			for {
				select {
				case <-ctx.Done():
					return
				case <-pulse:
					sendPulse()
				case results <- r:
					return
				}
			}
		}

		for i := 0; i < 5; i++ {
			select {
			case <-ctx.Done():
				return
			case <-pulse:
				sendPulse()
			case r := <-workGen:
				sendResult(r)
			}
		}

	}()

	return heartbeats, results
}

func doWorkWithHeartbeatBefore(ctx context.Context, numbers ...int) (<-chan interface{}, <-chan int) {
	heartbeats := make(chan interface{}, 1)
	results := make(chan int)

	go func() {
		defer close(heartbeats)
		defer close(results)

		time.Sleep(2 * time.Second)

		for _, n := range numbers {
			select {
			case heartbeats <- struct{}{}:
			default:
			}

			select {
			case <-ctx.Done():
				return
			case results <- n:
			}
		}
	}()

	return heartbeats, results
}

func doWorkWithHeartbeatBeforeWithPulseInterval(ctx context.Context, pulseInterval time.Duration, numbers ...int) (<-chan interface{}, <-chan int) {
	heartbeats := make(chan interface{}, 1)
	results := make(chan int)

	go func() {
		defer close(heartbeats)
		defer close(results)

		time.Sleep(2 * time.Second)
		pulse := time.Tick(pulseInterval)

	numLoop:
		for _, n := range numbers {
			for {
				select {
				case <-ctx.Done():
					return
				case <-pulse:
					select {
					case heartbeats <- struct{}{}:
					default:
					}
				case results <- n:
					continue numLoop
				}
			}
		}
	}()

	return heartbeats, results
}
