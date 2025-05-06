package main

import (
	"go-patterns/common/or"
	"log"
	"os"
	"time"
)

func main() {
	log.SetOutput(os.Stdout)
	log.SetFlags(log.Ltime | log.LUTC)

	doWork := func(done <-chan interface{}, _ time.Duration) <-chan interface{} {
		log.Println("ward: Hello, I'm irresponsible!")

		go func() {
			<-done
			log.Println("ward: I am halting.")
		}()

		return nil
	}

	doWorkWithSteward := newSteward(4*time.Second, doWork)

	done := make(chan interface{})

	time.AfterFunc(9*time.Second, func() {
		log.Println("main: halting steward and ward.")
		close(done)
	})

	for range doWorkWithSteward(done, 4*time.Second) {
	}
	log.Println("Done")
}

type startGoroutineFn func(
	done <-chan interface{},
	pulseInterval time.Duration,
) (heartbeat <-chan interface{})

func newSteward(
	timeout time.Duration,
	startGoroutine startGoroutineFn,
) startGoroutineFn {
	return func(done <-chan interface{}, pulseInterval time.Duration) <-chan interface{} {
		heartbeat := make(chan interface{})

		go func() {
			defer close(heartbeat)

			var wardDone chan interface{}
			var wardHeartbeat <-chan interface{}

			startWard := func() {
				wardDone = make(chan interface{})
				wardHeartbeat = startGoroutine(or.Or(done, wardDone), timeout/2)
			}
			startWard()

			pulse := time.Tick(pulseInterval)

		monitorLoop:
			for {
				timeoutSignal := time.After(timeout)

				for {
					select {
					case <-pulse:
						select {
						case heartbeat <- struct{}{}:
						default:
						}
					case <-wardHeartbeat:
						continue monitorLoop
					case <-timeoutSignal:
						log.Println("steward: ward unhealthy; restarting")
						close(wardDone)
						startWard()
						continue monitorLoop
					case <-done:
						return
					}
				}
			}
		}()

		return heartbeat
	}
}

func newStewardWithParams(done <-chan interface{}, ints ...int) (startGoroutineFn, <-chan interface{}) {
	intChanStream := make(chan (<-chan interface{}))
	intStream := bridge(done, intChanStream)

	doWork := func(
		done <-chan interface{},
		pulseInterval time.Duration,
	) <-chan interface{} {
		intStream := make(chan interface{})
		heartbeat := make(chan interface{})

		go func() {
			defer close(intStream)

			select {
			case intChanStream <- intStream:
			case <-done:
				return
			}

			pulse := time.Tick(pulseInterval)

			for {
			valueLoop:
				for _, intVal := range ints {
					if intVal < 0 {
						log.Printf("negative value: %v\n", intVal)
						return
					}

					for {
						select {
						case <-pulse:
							select {
							case heartbeat <- struct{}{}:
							default:
							}
						case intStream <- intVal:
							continue valueLoop
						case <-done:
							return
						}
					}
				}
			}

		}()

		return heartbeat
	}

	return doWork, intStream
}

func bridge(done <-chan interface{}, channelOfChannels <-chan <-chan interface{}) <-chan interface{} {
	result := make(chan interface{})

	go func() {
		defer close(result)

		for {
			var stream <-chan interface{}

			select {
			case <-done:
				return
			case maybeStream, ok := <-channelOfChannels:
				if !ok {
					return
				}
				stream = maybeStream
			}

			for val := range stream {
				select {
				case <-done:
				case result <- val:
				}
			}
		}
	}()

	return result
}
