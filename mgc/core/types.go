package core

import (
	"context"
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

// NOTE: TODO: should we duplicate this, or find a more generic package?
type Schema openapi3.Schema

func (s *Schema) VisitJSON(value any, opts ...openapi3.SchemaValidationOption) error {
	return (*openapi3.Schema)(s).VisitJSON(value, opts...)
}

// General interface that describes both Executor and Grouper
type Descriptor interface {
	Name() string
	Version() string
	Description() string
}

type DescriptorVisitor func(child Descriptor) (run bool, err error)

type Grouper interface {
	Descriptor
	VisitChildren(visitor DescriptorVisitor) (finished bool, err error)
	GetChildByName(name string) (child Descriptor, err error)
}

// contextKey is an unexported type for keys defined in this package.
// This prevents collisions with keys defined in other packages.
type contextKey string

// grouperContextKey is the key for sdk.Grouper values in Contexts. It is
// unexported; clients use NewGrouperContext() and GrouperFromContext()
// instead of using this key directly.
var grouperContextKey contextKey = "magalu.cloud/core/Grouper"

func NewGrouperContext(parent context.Context, group Grouper) context.Context {
	return context.WithValue(parent, grouperContextKey, group)
}

func GrouperFromContext(ctx context.Context) Grouper {
	if value, ok := ctx.Value(grouperContextKey).(Grouper); !ok {
		return nil
	} else {
		return value
	}
}

// Type comes from the Schema
type Value = any

// Type comes from the Schema
type Example = Value

type Executor interface {
	Descriptor
	ParametersSchema() *Schema
	ConfigsSchema() *Schema
	ResultSchema() *Schema
	Execute(context context.Context, parameters map[string]Value, configs map[string]Value) (result Value, err error)
}

func VisitAllExecutors(child Descriptor, path []string, visitExecutor func(executor Executor, path []string) (bool, error)) (bool, error) {
	if executor, ok := child.(Executor); ok {
		return visitExecutor(executor, path)
	} else if group, ok := child.(Grouper); ok {
		return group.VisitChildren(func(child Descriptor) (run bool, err error) {
			size := len(path)
			path = append(path, child.Name())
			run, err = VisitAllExecutors(child, path, visitExecutor)
			path = path[:size]

			return run, err
		})
	} else {
		return false, fmt.Errorf("child %v not group/executor", child)
	}
}

// Implement this interface in Executor()s that want to provide customized formatting of output.
// It's used by the command line interface (CLI) and possible other tools.
// This is only called if no other explicit formatting is desired
type ExecutorResultFormatter interface {
	Executor
	// NOTE: result is the converted value, such as primitives, map[string]any, []any...
	// Whenever using StaticExecute, it's *NOT* the ResultT (ie: struct)
	DefaultFormatResult(result Value) string
}

type executeFormat struct {
	Executor
	formatter func(result Value) string
}

func (o *executeFormat) DefaultFormatResult(result Value) string {
	return o.formatter(result)
}

var _ ExecutorResultFormatter = (*executeFormat)(nil)

// Wraps (embeds) an executor and add specific result formatting.
func NewExecuteFormat(
	executor Executor,
	formatter func(result Value) string,
) ExecutorResultFormatter {
	return &executeFormat{executor, formatter}
}
