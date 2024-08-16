package main

import (
	"context"
	"fmt"
	"go-patterns/common/generators"
	"go-patterns/common/or_done"
	"sync"
)

func main() {
	ctx := context.Background()
	wg := sync.WaitGroup{}

	channels := funOut(ctx, generators.TakeGenerator(ctx, generators.RepeatGenerator(ctx, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24), 23), 5)
	for i, ch := range channels {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for val := range ch {
				fmt.Println(fmt.Sprintf("channel %d: %d", i, val))
			}
		}()
	}

	wg.Wait()
}

func funOut(ctx context.Context, inChan <-chan int, chanNum int) []<-chan int {
	result := make([]<-chan int, 0, chanNum)

	for i := 0; i < chanNum; i++ {
		resultChan := make(chan int)
		result = append(result, resultChan)

		go func() {
			defer close(resultChan)

			for val := range ordone.OrDone(ctx, inChan) {
				resultChan <- val
			}
		}()
	}

	return result
}
