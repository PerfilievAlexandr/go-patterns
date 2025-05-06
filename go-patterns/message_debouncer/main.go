package main

import (
	"context"
	"fmt"
	"time"
)

func main() {
	ctx := context.Background()

	svc := service{}

	handler := NewMessageHandlerWithRateLimiter(svc, Config{
		RatePeriodMicroseconds: 1000,
		RequestsPerPeriod:      5,
	})

	err := handler.Send(ctx)
	if err != nil {
		panic(err)
	}
}

type Sender interface {
	Send(ctx context.Context) error
}

type service struct{}

func (r service) Send(_ context.Context) error {
	fmt.Println("send")
	return nil
}

type messageHandlerWithRateLimiter struct {
	service     Sender
	maxRequests int64
	ticker      *time.Ticker
}

type Config struct {
	RatePeriodMicroseconds int64
	RequestsPerPeriod      int64
}

func NewMessageHandlerWithRateLimiter(service Sender, config Config) Sender {
	return messageHandlerWithRateLimiter{
		service:     service,
		maxRequests: config.RequestsPerPeriod,
		ticker:      time.NewTicker(time.Duration(config.RatePeriodMicroseconds) * time.Microsecond),
	}
}

func (r messageHandlerWithRateLimiter) Send(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-r.ticker.C:
			requestCounter := 0

			for requestCounter < int(r.maxRequests) {
				err := r.service.Send(ctx)
				if err != nil {
					return err
				}

				requestCounter++
			}
		}
	}
}
