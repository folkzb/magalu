package core

type Linker interface {
	Name() string
	Description() string
	// Describes the additional parameters required by the created executor.
	//
	// This will match CreateExecutor().ParametersSchema()
	AdditionalParametersSchema() *Schema
	// Describes the additional configuration required by the created executor.
	//
	// This will match CreateExecutor().ConfigsSchema()
	AdditionalConfigsSchema() *Schema
	ResultSchema() *Schema
	// Create an executor based on a result.
	//
	// The returned executor will have ParametersSchema() matching AdditionalParametersSchema()
	// and ConfigsSchema() matching AdditionalConfigsSchema()
	CreateExecutor(originalResult Result) (exec Executor, err error)
}
