package core

import (
	"context"
	"fmt"
	"time"
)

type TerminatorExecutor interface {
	Executor
	ExecuteUntilTermination(context context.Context, parameters map[string]Value, configs map[string]Value, maxRetries int, interval time.Duration) (result Value, err error)
}

type executeTerminatorWithCheck struct {
	Executor
	defaultMaxRetries int
	defatultInterval  time.Duration
	checkTerminate    func(ctx context.Context, exec Executor, result Value) bool
}

type MaxRetriesExceededError struct {
	Result     Value
	maxRetries int
	interval   time.Duration
}

func (e MaxRetriesExceededError) Error() string {
	return fmt.Sprintf("maximum number of retries exceeded. Retries: %d, interval: %s", e.maxRetries, e.interval)
}

func (o *executeTerminatorWithCheck) Unwrap() Executor {
	return o.Executor
}

func (o *executeTerminatorWithCheck) ExecuteUntilTermination(context context.Context, parameters map[string]Value, configs map[string]Value, maxRetries int, interval time.Duration) (result Value, err error) {
	if interval <= 0 {
		interval = o.defatultInterval
	}
	if maxRetries <= 0 {
		maxRetries = o.defaultMaxRetries
	}

	for i := 0; i < maxRetries; i++ {
		result, err = o.Execute(context, parameters, configs)
		if err != nil {
			return result, err
		}
		if o.checkTerminate(context, o.Executor, result) {
			return result, nil
		}

		timer := time.NewTimer(interval)
		select {
		case <-context.Done():
			timer.Stop()
			return nil, context.Err()
		case <-timer.C:
		}
	}
	return result, MaxRetriesExceededError{Result: result, maxRetries: maxRetries, interval: interval}
}

var _ TerminatorExecutor = (*executeTerminatorWithCheck)(nil)

func NewTerminatorExecutorWithCheck(
	executor Executor,
	defaultMaxRetries int,
	defaultInterval time.Duration,
	checkTerminate func(ctx context.Context, exec Executor, result Value) bool,
) TerminatorExecutor {
	return &executeTerminatorWithCheck{executor, defaultMaxRetries, defaultInterval, checkTerminate}
}
