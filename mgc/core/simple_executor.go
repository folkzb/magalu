package core

import (
	"context"
	"errors"
	"fmt"
)

type Links map[string]Linker

func (l Links) AddLink(name string, target Linker) bool {
	_, ok := l[name]
	if !ok {
		l[name] = target
	}

	return !ok
}

type LinksSpecFn func() Links
type RelatedSpecFn func() map[string]Executor
type ExecutorSpecFn func(executor Executor, context context.Context, parameters Parameters, configs Configs) (result Result, err error)

type ExecutorSpec struct {
	DescriptorSpec
	ParametersSchema *Schema
	ConfigsSchema    *Schema
	ResultSchema     *Schema
	Links            LinksSpecFn
	Related          RelatedSpecFn
	Execute          ExecutorSpecFn
}

var errNil = errors.New("cannot be nil")

func (s *ExecutorSpec) Validate() (err error) {
	err = s.DescriptorSpec.Validate()
	if err != nil {
		return
	}

	if s.ParametersSchema == nil {
		return &ChainedError{"ParametersSchema", errNil}
	} else if s.ParametersSchema.Type != "object" {
		return &ChainedError{"ParametersSchema", fmt.Errorf("want object, got %q", s.ParametersSchema.Type)}
	}

	if s.ConfigsSchema == nil {
		return &ChainedError{"ConfigsSchema", errNil}
	} else if s.ConfigsSchema.Type != "object" {
		return &ChainedError{"ConfigsSchema", fmt.Errorf("want object, got %q", s.ConfigsSchema.Type)}
	}

	if s.ResultSchema == nil {
		return &ChainedError{"ResultSchema", errNil}
	}

	if s.Execute == nil {
		return &ChainedError{"Execute", errNil}
	}

	return nil
}

type SimpleExecutor struct {
	SimpleDescriptor

	parametersSchema *Schema
	configsSchema    *Schema
	resultSchema     *Schema
	links            LinksSpecFn
	related          RelatedSpecFn
	execute          ExecutorSpecFn
}

func NewSimpleExecutor(spec ExecutorSpec) *SimpleExecutor {
	err := spec.Validate()
	if err != nil {
		logger().Fatalw("invalid spec", "spec", spec, "err", err)
	}
	return &SimpleExecutor{
		SimpleDescriptor{spec.DescriptorSpec},
		spec.ParametersSchema,
		spec.ConfigsSchema,
		spec.ResultSchema,
		spec.Links,
		spec.Related,
		spec.Execute,
	}
}

func (e *SimpleExecutor) ParametersSchema() *Schema {
	return e.parametersSchema
}

func (e *SimpleExecutor) ConfigsSchema() *Schema {
	return e.configsSchema
}

func (e *SimpleExecutor) ResultSchema() *Schema {
	return e.resultSchema
}

func (e *SimpleExecutor) Links() Links {
	if e.links == nil {
		return nil
	}
	return e.links()
}

func (e *SimpleExecutor) Related() map[string]Executor {
	if e.related == nil {
		return nil
	}
	return e.related()
}

func (e *SimpleExecutor) Execute(context context.Context, parameters Parameters, configs Configs) (result Result, err error) {
	return e.execute(e, context, parameters, configs)
}

var _ Executor = (*SimpleExecutor)(nil)
