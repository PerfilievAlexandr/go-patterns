package main

import (
	"cmp"
	"context"
	"fmt"
	ordone "go-patterns/common/or_done"
	"time"
)

func MergeSortedChan[T cmp.Ordered](ctx context.Context, a, b <-chan T) <-chan T {
	result := make(chan T)

	go func() {
		defer close(result)

		aVal, aOk := <-ordone.OrDone(ctx, a)
		bVal, bOk := <-ordone.OrDone(ctx, b)

		for aOk && bOk {
			if aVal <= bVal {
				result <- aVal
				a = ordone.OrDone(ctx, a)
				aVal, aOk = <-a
			} else {
				result <- bVal
				b = ordone.OrDone(ctx, b)
				bVal, bOk = <-b
			}
		}

		for aOk {
			result <- aVal
			a = ordone.OrDone(ctx, a)
			aVal, aOk = <-a
		}

		for bOk {
			result <- bVal
			b = ordone.OrDone(ctx, b)
			bVal, bOk = <-b
		}

	}()

	return result
}

func main() {
	a := make(chan int)
	b := make(chan int)

	go func() {
		defer close(a)
		a <- 1
		a <- 2
		a <- 3
		time.Sleep(time.Second * 2)
		a <- 6
	}()

	go func() {
		defer close(b)
		b <- 2
		time.Sleep(time.Second)
		b <- 2
		b <- 3
		b <- 5
	}()

	resCh := MergeSortedChan(context.Background(), a, b)
	for val := range resCh {
		fmt.Println(val)
	}
}
