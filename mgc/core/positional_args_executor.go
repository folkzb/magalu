package core

import "context"

type PositionalArgsExecutor interface {
	Executor
	PositionalArgs() []string
}

type positionalArgsExecutor struct {
	Executor
	args []string
}

func NewPositionalArgsExecutor(exec Executor, args []string) PositionalArgsExecutor {
	return &positionalArgsExecutor{exec, args}
}

func (p *positionalArgsExecutor) PositionalArgs() []string {
	return p.args
}

func (p *positionalArgsExecutor) Execute(ctx context.Context, parameters Parameters, configs Configs) (Result, error) {
	result, err := p.Executor.Execute(ctx, parameters, configs)
	return ExecutorWrapResult(p, result, err)
}

func (p *positionalArgsExecutor) Unwrap() Executor {
	return p.Executor
}

var _ Executor = (*positionalArgsExecutor)(nil)
var _ ExecutorWrapper = (*positionalArgsExecutor)(nil)
var _ PositionalArgsExecutor = (*positionalArgsExecutor)(nil)
