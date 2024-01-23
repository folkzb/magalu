package blueprint

import (
	"errors"

	"go.uber.org/zap"
	"magalu.cloud/core"
	schemaPkg "magalu.cloud/core/schema"
)

func newChild(spec *childSpec, logger *zap.SugaredLogger, refResolver *core.BoundRefPathResolver) (core.Descriptor, error) {
	// If the spec specifies a $ref, the we solve the reference and
	// use it to populate the spec. This way we can copy a refered executor or grouper
	// and make changes on it by setting in on the blueprint
	if spec.Ref != "" {
		resolved, err := refResolver.Resolve(spec.Ref)
		if err != nil {
			return nil, err
		}
		resolvedDesc, ok := resolved.(core.Descriptor)
		if !ok {
			return nil, errors.New("ref for child must be a descriptor")
		}
		populateEmptyFrom(resolvedDesc, spec, core.RefPath(spec.Ref))
	}
	if !spec.grouperSpec.isEmpty() {
		return newGrouper(spec, logger, refResolver)
	}
	return newExecutor(spec, logger, refResolver)
}

func populateEmptyFrom(newSpec core.Descriptor, spec *childSpec, ref core.RefPath) {
	if spec.Name == "" {
		spec.Name = newSpec.Name()
	}
	if spec.Version == "" {
		spec.Version = newSpec.Version()
	}
	if spec.Description == "" {
		spec.Description = newSpec.Description()
	}
	if spec.Summary == "" {
		spec.Summary = newSpec.Summary()
	}
	if spec.IsInternal == nil {
		temp := newSpec.IsInternal()
		spec.IsInternal = &temp
	}
	if spec.Scopes == nil {
		spec.Scopes = newSpec.Scopes()
	}
	populateExecutor(spec, newSpec, ref)
	populateGrouper(spec, newSpec, ref)
}

func populateExecutor(spec *childSpec, newSpec core.Descriptor, ref core.RefPath) {
	executor, ok := newSpec.(core.Executor)
	if !ok {
		return
	}

	spec.linkers = executor.Links()

	if spec.parametersSchema == nil && spec.ParametersSchema == nil {
		spec.ParametersSchema = schemaPkg.NewSchemaRef("", executor.ParametersSchema())
	}
	if spec.configsSchema == nil && spec.ConfigsSchema == nil {
		spec.ConfigsSchema = schemaPkg.NewSchemaRef("", executor.ConfigsSchema())
	}

	if spec.resultSchema == nil && spec.ResultSchema == nil {
		spec.ResultSchema = schemaPkg.NewSchemaRef("", executor.ResultSchema())
	}

	if spec.Steps == nil {
		parameters := map[string]string{}
		for paramName := range executor.ParametersSchema().Properties {
			parameters[paramName] = "$.parameters." + paramName
		}
		configs := map[string]string{}
		for configName := range executor.ConfigsSchema().Properties {
			configs[configName] = "$.configs." + configName
		}
		spec.Steps = []*executeStep{
			{
				Target:     ref,
				Parameters: parameters,
				Configs:    configs,
			},
		}
	}
}

func populateGrouper(spec *childSpec, newSpec core.Descriptor, ref core.RefPath) {
	grouper, ok := newSpec.(core.Grouper)
	if !ok {
		return
	}

	childrenCopy := []*childSpec{}
	_, err := grouper.VisitChildren(func(child core.Descriptor) (run bool, err error) {
		for _, specChild := range spec.Children {
			if specChild.Name == child.Name() {
				return true, nil
			}
		}

		child_spec := &childSpec{}
		populateEmptyFrom(child, child_spec, ref.Add(child.Name()))

		childrenCopy = append(childrenCopy, child_spec)

		return true, nil
	})

	if err != nil {
		return
	}

	spec.Children = childrenCopy
}
