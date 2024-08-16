package generators

import (
	"context"
)

func RepeatGenerator(ctx context.Context, values ...int) <-chan int {
	result := make(chan int)

	go func() {
		defer close(result)

		for {
			for _, value := range values {
				select {
				case <-ctx.Done():
				case result <- value:
				}
			}
		}
	}()

	return result
}

func RepeatFuncGenerator(ctx context.Context, functions ...func() int) <-chan int {
	result := make(chan int)

	go func() {
		defer close(result)

		for {
			for _, function := range functions {
				select {
				case <-ctx.Done():
				case result <- function():
				}
			}
		}
	}()

	return result
}

func TakeGenerator(ctx context.Context, inputCh <-chan int, count int) <-chan int {
	result := make(chan int)

	go func() {
		defer close(result)

		for i := 0; i < count; i++ {
			select {
			case <-ctx.Done():
			case result <- <-inputCh:
			}
		}
	}()

	return result
}
