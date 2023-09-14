package core

import (
	"context"
	"fmt"
	"time"
)

type TerminatorExecutor interface {
	Executor
	ExecuteUntilTermination(context context.Context, parameters map[string]Value, configs map[string]Value) (result Value, err error)
}

type FailedTerminationError struct {
	Result  Value
	Message string
}

func (e FailedTerminationError) Error() string {
	return e.Message
}

type executeTerminatorWithCheck struct {
	Executor
	maxRetries     int
	interval       time.Duration
	checkTerminate func(ctx context.Context, exec Executor, result Value) (terminated bool, err error)
}

func (o *executeTerminatorWithCheck) Unwrap() Executor {
	return o.Executor
}

func (o *executeTerminatorWithCheck) ExecuteUntilTermination(context context.Context, parameters map[string]Value, configs map[string]Value) (result Value, err error) {
	for i := 0; i < o.maxRetries; i++ {
		result, err = o.Execute(context, parameters, configs)
		if err != nil {
			return result, err
		}
		terminated, err := o.checkTerminate(context, o.Executor, result)
		if err != nil {
			return result, err
		}
		if terminated {
			return result, nil
		}

		timer := time.NewTimer(o.interval)
		select {
		case <-context.Done():
			timer.Stop()
			return nil, context.Err()
		case <-timer.C:
		}
	}

	msg := fmt.Sprintf("maximum number of retries exceeded. Retries: %d, interval: %s", o.maxRetries, o.interval)
	return result, FailedTerminationError{Result: result, Message: msg}
}

var _ TerminatorExecutor = (*executeTerminatorWithCheck)(nil)

// Execute the operation and check the results until it's considered terminated.
// The executor will wait `interval` between retries, executing up to `maxRetries`
func NewTerminatorExecutorWithCheck(
	executor Executor,
	maxRetries int,
	interval time.Duration,
	checkTerminate func(ctx context.Context, exec Executor, result Value) (terminated bool, err error),
) TerminatorExecutor {
	return &executeTerminatorWithCheck{executor, maxRetries, interval, checkTerminate}
}
