package core

import (
	"context"
	"io"
	"mime/multipart"
)

type ResultSource struct {
	Executor   Executor
	Context    context.Context
	Parameters map[string]Value
	Configs    map[string]Value
}

type Result interface {
	// What was used to produce this result
	Source() ResultSource
}

type ResultWithValue interface {
	Result
	Schema() *Schema
	// Check value against schema, reports any error
	ValidateSchema() error

	Value() Value
}

type ResultWithReader interface {
	Result
	Reader() io.Reader
}

type ResultWithMultipart interface {
	Result
	Multipart() *multipart.Part
}

type ResultWrapper interface {
	Result
	Unwrap() Result
}

func ResultAs[T Result](result Result) (T, bool) {
	var zeroT T

	for {
		if t, ok := result.(T); ok {
			return t, true
		}

		if u, ok := result.(ResultWrapper); ok {
			result = u.Unwrap()
		} else {
			break
		}
	}

	return zeroT, false
}

type SimpleResult struct {
	source ResultSource
	schema *Schema
	value  Value
}

func NewSimpleResult(source ResultSource, schema *Schema, value Value) *SimpleResult {
	return &SimpleResult{source, schema, value}
}

func (s SimpleResult) Source() ResultSource {
	return s.source
}

func (s SimpleResult) Schema() *Schema {
	return s.schema
}

func (s SimpleResult) ValidateSchema() error {
	return s.schema.VisitJSON(s.value)
}

func (s SimpleResult) Value() Value {
	return s.value
}

var _ ResultWithValue = (*SimpleResult)(nil)

type resultWithOriginalSource struct {
	Result
	originalSource ResultSource
}

func (o resultWithOriginalSource) Source() ResultSource {
	return o.originalSource
}

func (o resultWithOriginalSource) Unwrap() Result {
	return o.Result
}

var _ ResultWrapper = (*resultWithOriginalSource)(nil)

// Wraps (embeds) a result and overrides the original source.
func NewResultWithOriginalSource(originalSource ResultSource, result Result) *resultWithOriginalSource {
	return &resultWithOriginalSource{result, originalSource}
}

func NewResultWithOriginalExecutor(originalExecutor Executor, result Result) Result {
	originalSource := result.Source()
	if originalSource.Executor == originalExecutor {
		return result
	}
	originalSource.Executor = originalExecutor
	return NewResultWithOriginalSource(originalSource, result)
}

// Implement this interface in Result that want to provide customized formatting of output.
// It's used by the command line interface (CLI) and possible other tools.
// This is only called if no other explicit formatting is desired
type ResultWithDefaultFormatter interface {
	ResultWithValue
	DefaultFormatter() string
}

type resultWithDefaultFormatter struct {
	ResultWithValue
	formatter string
}

func (o resultWithDefaultFormatter) DefaultFormatter() string {
	return o.formatter
}

func (o resultWithDefaultFormatter) Unwrap() Result {
	return o.ResultWithValue
}

var _ ResultWithDefaultFormatter = (*resultWithDefaultFormatter)(nil)
var _ ResultWrapper = (*resultWithDefaultFormatter)(nil)

// Wraps (embeds) a result and add specific result formatting.
func NewResultWithDefaultFormatter(
	result ResultWithValue,
	formatter string,
) ResultWithDefaultFormatter {
	return &resultWithDefaultFormatter{result, formatter}
}

// Implement this interface in Results that want to provide default output options.
// It's used by the command line interface (CLI) and possible other tools.
// This is only called if no other explicit options are desired
type ResultWithDefaultOutputOptions interface {
	ResultWithValue
	// The return should be in the same format as CLI -o "VALUE"
	// example: "yaml" or "table=COL:$.path.to[*].element,OTHERCOL:$.path.to[*].other"
	DefaultOutputOptions() string
}

type resultWithDefaultOutputOptions struct {
	ResultWithValue
	outputOptions string
}

func (o resultWithDefaultOutputOptions) DefaultOutputOptions() string {
	return o.outputOptions
}

func (o resultWithDefaultOutputOptions) Unwrap() Result {
	return o.ResultWithValue
}

var _ ResultWithDefaultOutputOptions = (*resultWithDefaultOutputOptions)(nil)
var _ ResultWrapper = (*resultWithDefaultFormatter)(nil)

// Wraps (embeds) a result and add specific result default output options getter.
func NewResultWithDefaultOutputOptions(
	result ResultWithValue,
	outputOptions string,
) ResultWithDefaultOutputOptions {
	return &resultWithDefaultOutputOptions{result, outputOptions}
}
