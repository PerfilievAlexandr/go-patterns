package main

import (
	"context"
	"testing"
)

func Test_doWorkWithHeartbeatBefore(t *testing.T) {
	ctx := context.Background()
	intSlice := []int{1, 2, 3, 4, 5}

	heartbeats, results := doWorkWithHeartbeatBefore(ctx, intSlice...)

	<-heartbeats

	i := 0
	for r := range results {
		if expected := intSlice[i]; r != expected {
			t.Errorf("index %v: expected %v, but received %v,", i, expected, r)
		}
		i++
	}
}
