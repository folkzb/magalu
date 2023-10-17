package core

import "context"

var noOpExecutorInstance Executor = nil

func NoOpExecutor() Executor {
	if noOpExecutorInstance == nil {
		noOpExecutorInstance = NewStaticExecute(
			"",
			"",
			"",
			func(context context.Context, params Parameters, configs Configs) (result Value, err error) {
				return nil, nil
			},
		)
	}
	return noOpExecutorInstance
}
