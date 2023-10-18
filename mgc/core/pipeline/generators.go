package pipeline

import "context"

// Generates integers in interval [0,n)
func RangeGenerator(ctx context.Context, n int) <-chan int {
	c := make(chan int)
	go func() {
		defer close(c)
		for i := 0; i < n; i++ {
			iCopy := i
			select {
			case <-ctx.Done():
				return
			case c <- iCopy:
			}
		}
	}()
	return c
}

// Sends all entries of slice to a channel
func SliceItemGenerator[T any](ctx context.Context, slice []T) <-chan T {
	c := make(chan T)
	go func() {
		defer close(c)
		for _, item := range slice {
			itemCopy := item
			select {
			case <-ctx.Done():
				return
			case c <- itemCopy:
			}
		}
	}()
	return c
}
