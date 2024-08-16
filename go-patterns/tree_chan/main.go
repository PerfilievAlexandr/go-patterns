package main

import (
	"context"
	"fmt"
	"go-patterns/common/generators"
	ordone "go-patterns/common/or_done"
)

func main() {
	ctx := context.Background()

	done := make(chan int)
	defer close(done)
	out1, out2 := treeChan(ctx, generators.TakeGenerator(ctx, generators.RepeatGenerator(ctx, 1, 2), 4))
	for val1 := range out1 {
		fmt.Printf("out1: %v, out2: %v\n", val1, <-out2)
	}
}

func treeChan(ctx context.Context, inChan <-chan int) (<-chan int, <-chan int) {
	outCh1 := make(chan int)
	outCh2 := make(chan int)

	go func() {
		defer func() {
			close(outCh1)
			close(outCh2)
		}()

		for inVal := range ordone.OrDone(ctx, inChan) {
			var outCh1, outCh2 = outCh1, outCh2
			for i := 0; i < 2; i++ {
				select {
				case <-ctx.Done():
				case outCh1 <- inVal:
					outCh1 = nil
				case outCh2 <- inVal:
					outCh2 = nil
				}
			}
		}
	}()

	return outCh1, outCh2
}
