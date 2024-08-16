package main

import (
	"context"
	"fmt"
)

func main() {
	ctx := context.Background()

	channelWithChannels := make(chan (<-chan int))

	go func() {
		defer close(channelWithChannels)

		for i := 0; i < 10; i++ {
			ch := make(chan int, 1)
			ch <- i
			close(ch)
			channelWithChannels <- ch
		}
	}()

	for val := range bridge(ctx, channelWithChannels) {
		fmt.Printf("%v ", val)
	}

}

func bridge(ctx context.Context, channelOfChannels <-chan <-chan int) <-chan int {
	result := make(chan int)

	go func() {
		defer close(result)

		for {
			var stream <-chan int

			select {
			case <-ctx.Done():
				return
			case maybeStream, ok := <-channelOfChannels:
				if !ok {
					return
				}
				stream = maybeStream
			}

			for val := range stream {
				select {
				case <-ctx.Done():
				case result <- val:
				}
			}
		}
	}()

	return result
}
