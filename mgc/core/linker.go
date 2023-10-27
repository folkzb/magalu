package core

import (
	"golang.org/x/exp/slices"
	mgcSchemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

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

type Links map[string]Linker

type simpleLink struct {
	name                          string
	description                   string
	owner                         Executor
	target                        Executor
	getAdditionalParametersSchema func() *Schema
	getAdditionalConfigsSchema    func() *Schema
}

var _ Linker = (*simpleLink)(nil)

func NewSimpleLink(owner Executor, target Executor) *simpleLink {
	l := &simpleLink{
		name:        target.Name(),
		description: target.Description(),
		owner:       owner,
		target:      target,
	}

	l.getAdditionalParametersSchema = utils.NewLazyLoader[*Schema](
		func() *Schema {
			return additionalProps(l.target.ParametersSchema(), l.owner.ResultSchema(), l.owner.ParametersSchema())
		},
	)

	l.getAdditionalConfigsSchema = utils.NewLazyLoader[*Schema](
		func() *Schema {
			return additionalProps(l.target.ConfigsSchema(), l.owner.ResultSchema(), l.owner.ConfigsSchema())
		},
	)

	return l
}

func (l *simpleLink) Name() string {
	return l.name
}

func (l *simpleLink) Description() string {
	return l.description
}

func additionalProps(target *Schema, sources ...*Schema) *Schema {
	additional := map[string]*Schema{}
	required := []string{}

TargetLoop:
	for propName, propRef := range target.Properties {
		for _, source := range sources {
			if _, ok := source.Properties[propName]; ok {
				continue TargetLoop
			}
		}

		additional[propName] = (*Schema)(propRef.Value)
		if slices.Contains(target.Required, propName) {
			required = append(required, propName)
		}
	}

	return mgcSchemaPkg.NewObjectSchema(additional, required)
}

func (l *simpleLink) AdditionalParametersSchema() *Schema {
	return l.getAdditionalParametersSchema()
}

func (l *simpleLink) AdditionalConfigsSchema() *Schema {
	return l.getAdditionalConfigsSchema()
}

func (l *simpleLink) ResultSchema() *Schema {
	return l.target.ResultSchema()
}

func injectValues(dst map[string]any, schema *Schema, sources ...map[string]any) {
SchemaLoop:
	for propName, propSchema := range schema.Properties {
		for _, source := range sources {
			if value, ok := source[propName]; ok {
				if err := propSchema.Value.VisitJSON(value); err != nil {
					// TODO: Should this be an error?
					continue
				}
				dst[propName] = value
				continue SchemaLoop
			}
		}
	}
}

func (l *simpleLink) CreateExecutor(originalResult Result) (Executor, error) {
	preparedParams := Parameters{}
	preparedConfigs := Configs{}

	var resultMap map[string]any
	if result, ok := ResultAs[ResultWithValue](originalResult); ok {
		resultMap, _ = result.Value().(map[string]any)
	}

	injectValues(preparedParams, l.target.ParametersSchema(), resultMap, originalResult.Source().Parameters)
	injectValues(preparedConfigs, l.target.ConfigsSchema(), resultMap, originalResult.Source().Configs)

	var exec LinkExecutor = NewLinkExecutor(l.target, preparedParams, preparedConfigs, l.AdditionalParametersSchema(), l.AdditionalConfigsSchema())

	if _, ok := ExecutorAs[ConfirmableExecutor](l.target); ok {
		exec = NewLinkConfirmableExecutor(exec)
	}

	if _, ok := ExecutorAs[TerminatorExecutor](l.target); ok {
		exec = NewLinkTerminatorExecutor(exec)
	}

	return exec, nil
}
