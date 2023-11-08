package pipeline

import "context"

func SliceItemConsumer[S ~[]T, T any](ctx context.Context, inputChan <-chan T) (result S, err error) {
	for input := range inputChan {
		select {
		case <-ctx.Done():
			return result, context.Cause(ctx)
		default:
			result = append(result, input)
		}
	}

	return result, context.Cause(ctx)
}

func SliceItemLimitedConsumer[S ~[]T, T any](ctx context.Context, limit int, inputChan <-chan T) (result S, err error) {
	for input := range inputChan {
		select {
		case <-ctx.Done():
			return result, context.Cause(ctx)
		default:
			result = append(result, input)

			if limit > 0 && len(result) >= limit {
				return result, context.Cause(ctx)
			}
		}
	}

	return result, context.Cause(ctx)
}
