package main

import (
	"context"
	"fmt"
	"go-patterns/common/generators"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

func main() {
	ctx := context.Background()

	random := func() int {
		return rand.Intn(50000000000)
	}

	start := time.Now()

	// with one core in work
	//for prime := range takeGenerator(ctx, sumChan(ctx, repeatFuncGenerator(ctx, random)), 3) {
	//	fmt.Printf("\t%d\n", prime)
	//}

	// with all cores in work
	numCPU := runtime.NumCPU()
	summers := make([]<-chan int, numCPU)
	for i := 0; i < numCPU; i++ {
		summers[i] = sumChan(ctx, generators.RepeatFuncGenerator(ctx, random))
	}
	for prime := range generators.TakeGenerator(ctx, funIn(ctx, summers...), 3) {
		fmt.Printf("\t%d\n", prime)
	}

	fmt.Printf("Search took: %v", time.Since(start))
}

func funIn(ctx context.Context, channels ...<-chan int) <-chan int {
	result := make(chan int)
	wg := sync.WaitGroup{}

	multiplex := func(ch <-chan int) {
		defer wg.Done()
		for val := range ch {
			select {
			case <-ctx.Done():
				return
			case result <- val:
			}
		}
	}

	wg.Add(len(channels))
	for _, channel := range channels {
		go multiplex(channel)
	}

	go func() {
		wg.Wait()
		close(result)
	}()

	return result
}

func sumChan(ctx context.Context, inputCh <-chan int) <-chan int {
	result := make(chan int)

	go func() {
		defer close(result)

		for inputVal := range inputCh {
			select {
			case <-ctx.Done():
				return
			default:
				var sum int
				for i := 0; i < inputVal; i++ {
					sum += i
				}
				result <- sum
			}
		}
	}()

	return result
}
