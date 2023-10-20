package pipeline_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"magalu.cloud/core/pipeline"
)

func TestSliceItemConsumer(t *testing.T) {
	ctx := context.Background()
	ch := make(chan int, 3)

	expected := []int{1, 2, 3}
	go func() {
		defer close(ch)
		for _, item := range expected {
			ch <- item
		}

	}()

	result, err := pipeline.SliceItemConsumer[[]int](ctx, ch)
	if err != nil {
		t.Errorf("Did not expect Consumer to fail")
	}
	if !reflect.DeepEqual(result, expected) {
		t.Errorf("Expected Consumer to generate %v, got %v", expected, result)
	}
}

func TestSliceItemConsumerError(t *testing.T) {
	ctx, cancel := context.WithCancelCause(context.Background())
	ch := make(chan int, 3)

	errorStr := "Received 3"

	expected := []int{1, 2, 3}
	go func() {
		defer close(ch)
		for _, item := range expected {
			if item == 3 {
				// Dummy error for no reason just to trigger
				cancel(errors.New(errorStr))
			}
			ch <- item
		}

	}()

	_, err := pipeline.SliceItemConsumer[[]int](ctx, ch)
	if err == nil {
		t.Errorf("Expected Consumer to return error")
	}
	if err.Error() != errorStr {
		t.Errorf("Expected error to be `%#v`, found `%#v`", errors.New(errorStr), err)
	}
}
