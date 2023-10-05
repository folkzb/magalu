package pipeline

// Generates integers in interval [0,n)
func RangeGenerator(n int) <-chan int {
	c := make(chan int)
	go func() {
		defer close(c)
		for i := 0; i < n; i++ {
			iCopy := i
			c <- iCopy
		}
	}()
	return c
}

// Sends all entries of slice to a channel
func SliceItemGenerator[T any](slice []T) <-chan T {
	c := make(chan T, len(slice))
	go func() {
		defer close(c)
		for _, item := range slice {
			itemCopy := item
			c <- itemCopy
		}
	}()
	return c
}
