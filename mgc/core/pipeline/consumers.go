package pipeline

import "context"

func SliceItemConsumer[T any](ctx context.Context, inputChan <-chan T) (result []T, err error) {
	for input := range inputChan {
		select {
		case <-ctx.Done():
			return result, context.Cause(ctx)
		default:
			result = append(result, input)
		}
	}

	return result, nil
}
