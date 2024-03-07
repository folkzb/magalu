package core

import (
	"context"
	"fmt"
	"time"
)

type TerminatorExecutor interface {
	Executor
	ExecuteUntilTermination(context context.Context, parameters Parameters, configs Configs) (result Result, err error)
}

type FailedTerminationError struct {
	Result  Result
	Message string
}

func (e FailedTerminationError) Error() string {
	return e.Message
}

type executeTerminatorWithCheck struct {
	Executor
	maxRetries     int
	interval       time.Duration
	checkTerminate func(ctx context.Context, exec Executor, result ResultWithValue) (terminated bool, err error)
}

func (o *executeTerminatorWithCheck) Unwrap() Executor {
	return o.Executor
}

func (o *executeTerminatorWithCheck) Execute(ctx context.Context, parameters Parameters, configs Configs) (result Result, err error) {
	result, err = o.Executor.Execute(ctx, parameters, configs)
	return ExecutorWrapResult(o, result, err)
}

func (o *executeTerminatorWithCheck) ExecuteUntilTermination(context context.Context, parameters Parameters, configs Configs) (result Result, err error) {
	result, err = o.executeUntilTermination(context, parameters, configs)
	return ExecutorWrapResult(o, result, err)
}

func (o *executeTerminatorWithCheck) executeUntilTermination(context context.Context, parameters Parameters, configs Configs) (result Result, err error) {
	var exec func() (Result, error)
	if tExec, ok := ExecutorAs[TerminatorExecutor](o.Unwrap()); ok {
		exec = func() (result Result, err error) {
			result, err = tExec.ExecuteUntilTermination(context, parameters, configs)
			return ExecutorWrapResult(o, result, err)
		}
	} else {
		exec = func() (result Result, err error) {
			return o.Execute(context, parameters, configs)
		}
	}

	for i := 0; i < o.maxRetries; i++ {
		result, err = exec()
		if err != nil {
			return result, err
		}
		resultWithValue, ok := ResultAs[ResultWithValue](result)
		if !ok {
			return result, fmt.Errorf("result does not have a value")
		}
		terminated, err := o.checkTerminate(context, o.Executor, resultWithValue)
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
var _ ExecutorWrapper = (*executeTerminatorWithCheck)(nil)

// Execute the operation and check the results until it's considered terminated.
// The executor will wait `interval` between retries, executing up to `maxRetries`
func NewTerminatorExecutorWithCheck(
	executor Executor,
	maxRetries int,
	interval time.Duration,
	checkTerminate func(ctx context.Context, exec Executor, result ResultWithValue) (terminated bool, err error),
) TerminatorExecutor {
	return &executeTerminatorWithCheck{executor, maxRetries, interval, checkTerminate}
}
