package core

import "context"

type HumanIdentifiableFieldsExecutor interface {
	Executor
	HumanIdentifiableFields() []string
}

type humanIdentifiableFieldsExecutor struct {
	Executor
	humanIdentifiableFields []string
}

func NewHumanIdentifiableFieldsExecutor(exec Executor, humanIdentifierFields []string) Executor {
	return &humanIdentifiableFieldsExecutor{exec, humanIdentifierFields}
}

func (e *humanIdentifiableFieldsExecutor) HumanIdentifiableFields() []string {
	return e.humanIdentifiableFields
}

func (e *humanIdentifiableFieldsExecutor) Execute(ctx context.Context, params Parameters, configs Configs) (Result, error) {
	result, err := e.Executor.Execute(ctx, params, configs)
	return ExecutorWrapResult(e, result, err)
}

func (e *humanIdentifiableFieldsExecutor) Unwrap() Executor {
	return e.Executor
}

var _ HumanIdentifiableFieldsExecutor = (*humanIdentifiableFieldsExecutor)(nil)
var _ ExecutorWrapper = (*humanIdentifiableFieldsExecutor)(nil)
