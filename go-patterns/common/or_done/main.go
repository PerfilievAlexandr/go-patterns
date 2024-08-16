package ordone

import (
	"context"
)

//func main() {
//	ctx, cancel := context.WithCancel(context.Background())
//	wg := sync.WaitGroup{}
//
//	go func() {
//		time.Sleep(time.Second * 4)
//		cancel()
//	}()
//
//	wg.Add(1)
//	go func() {
//		for chVal := range OrDone(ctx, generators.TakeGenerator(ctx, generators.RepeatGenerator(ctx, 1), 10)) {
//			fmt.Println(chVal)
//		}
//
//		wg.Done()
//		fmt.Println("exit")
//	}()
//
//	wg.Wait()
//}

func OrDone(ctx context.Context, ch <-chan int) <-chan int {
	result := make(chan int)

	go func() {
		defer close(result)

		for {
			select {
			case <-ctx.Done():
				return
			case val, ok := <-ch:
				if !ok {
					return
				}

				select {
				case <-ctx.Done():
				case result <- val:
				}
			}
		}
	}()

	return result
}
