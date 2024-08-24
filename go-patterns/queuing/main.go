package main

import (
	"context"
	"fmt"
	"go-patterns/common/generators"
	"time"
)

func main() {
	ctx := context.Background()

	zeros := generators.TakeGenerator(ctx, generators.RepeatGenerator(ctx, 1), 3)
	short := generators.SleepGenerator(ctx, zeros, 1*time.Second)
	long := generators.SleepGenerator(ctx, short, 4*time.Second)

	fmt.Println(<-long)
}
