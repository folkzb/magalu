package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

var errGeneric = errors.New("Error")

func withError(ctx context.Context) (bool, error) {
	time.Sleep(1 * time.Second)
	return false, errGeneric
}

func withSuccess(ctx context.Context) (bool, error) {
	time.Sleep(1 * time.Second)
	return true, nil
}

func exceedMaxRetries(ctx context.Context) (bool, error) {
	time.Sleep(1 * time.Second)
	return false, nil
}

type terminatorExecutorTestCase struct {
	name           string
	executor       Executor
	maxRetries     int
	interval       time.Duration
	checkTerminate func(ctx context.Context, exec Executor, result ResultWithValue) (terminated bool, err error)
	expectedError  error
	expectedValue  bool
}

func TestExecuteTerminatorWithCheck(t *testing.T) {

	ct := func(ctx context.Context, exec Executor, result ResultWithValue) (terminated bool, err error) {
		if exec.Name() == "ExceedMaxRetries" {
			return false, err
		}
		return true, err
	}

	tests := []terminatorExecutorTestCase{
		{
			maxRetries:     3,
			interval:       1 * time.Second,
			checkTerminate: ct,
			expectedValue:  true,
			expectedError:  nil,
			executor: NewStaticExecuteSimple(
				DescriptorSpec{
					Name:        "Success",
					Description: "Success",
				},
				withSuccess),
		},
		{
			maxRetries:     3,
			interval:       1 * time.Second,
			checkTerminate: ct,
			expectedValue:  false,
			expectedError:  errGeneric,
			executor: NewStaticExecuteSimple(
				DescriptorSpec{
					Name:        "Error",
					Description: "Error",
				},
				withError),
		},
		{
			maxRetries:     3,
			interval:       1 * time.Second,
			checkTerminate: ct,
			expectedValue:  false,
			expectedError:  errGeneric,
			executor: NewStaticExecuteSimple(
				DescriptorSpec{
					Name:        "ExceedMaxRetries",
					Description: "ExceedMaxRetries",
				},
				exceedMaxRetries),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			executor := &executeTerminatorWithCheck{
				Executor:       tc.executor,
				maxRetries:     tc.maxRetries,
				interval:       tc.interval,
				checkTerminate: tc.checkTerminate,
			}

			ctx := context.Background()
			parameters := make(map[string]interface{})
			configs := make(map[string]interface{})
			var result interface{} = false

			exeRes, err := executor.ExecuteUntilTermination(ctx, parameters, configs)
			resWV, hasValue := ResultAs[ResultWithValue](exeRes)
			if hasValue {
				result = resWV.Value()
			}

			if _, ok := err.(FailedTerminationError); ok == false && err != tc.expectedError && result != tc.expectedValue {
				t.Errorf("expected err == %s, found: %s", tc.expectedError.Error(), err.Error())
			}
		})
	}

}
