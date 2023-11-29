package blueprint

import (
	"context"
	"errors"
	"fmt"

	"slices"

	"github.com/PaesslerAG/gval"
	"magalu.cloud/core"
	schemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

type executeStep struct {
	Id              string            `json:"id"`
	IfCondition     string            `json:"if,omitempty"`
	Target          core.RefPath      `json:"target"`
	WaitTermination bool              `json:"waitTermination,omitempty"`
	RetryUntil      *retryUntil       `json:"retryUntil,omitempty"`
	Check           *checkSpec        `json:"check,omitempty"`
	Parameters      map[string]string `json:"parameters,omitempty"`
	Configs         map[string]string `json:"configs,omitempty"`

	// Materialized/Parsed values based on the JSON fields. They are populated in validate():

	ifJSONPath         gval.Evaluable
	executor           core.Executor
	parametersJSONPath map[string]gval.Evaluable
	configsJSONPath    map[string]gval.Evaluable
}

const defaultIfJSONPathText = "$.last == null || $.last.error == null"

var defaultIfJSONPath = func() gval.Evaluable {
	jp, err := utils.NewJsonPath(defaultIfJSONPathText)
	if err != nil {
		panic(fmt.Sprintf("invalid defaultIfJSONPath: %s", err))
	}
	return jp
}()

func (e *executeStep) validate() (err error) {
	if e.IfCondition == "" {
		e.IfCondition = defaultIfJSONPathText
		e.ifJSONPath = defaultIfJSONPath
	} else {
		e.ifJSONPath, err = utils.NewJsonPath(e.IfCondition)
		if err != nil {
			return &core.ChainedError{Name: "if", Err: err}
		}
	}

	if e.Target == "" {
		return errors.New("missing target")
	}

	if e.parametersJSONPath == nil {
		m := make(map[string]gval.Evaluable)
		for k, v := range e.Parameters {
			eval, err := utils.NewJsonPath(v)
			if err != nil {
				return &core.ChainedError{Name: "parameters", Err: &core.ChainedError{Name: k, Err: err}}
			}
			m[k] = eval
		}
		e.parametersJSONPath = m
	}

	if e.configsJSONPath == nil {
		m := make(map[string]gval.Evaluable)
		for k, v := range e.Configs {
			eval, err := utils.NewJsonPath(v)
			if err != nil {
				return &core.ChainedError{Name: "configs", Err: &core.ChainedError{Name: k, Err: err}}
			}
			m[k] = eval
		}
		e.configsJSONPath = m
	}

	if err = e.RetryUntil.validate(); err != nil {
		return &core.ChainedError{Name: "retryUntil", Err: err}
	}

	if err = e.Check.validate(); err != nil {
		return &core.ChainedError{Name: "check", Err: err}
	}

	return nil
}

func (e *executeStep) resolve(refResolver *core.BoundRefPathResolver) (err error) {
	e.executor, err = core.ResolveExecutorPath(refResolver, e.Target)
	return
}

func (e *executeStep) shouldExecute(jsonPathDocument map[string]any) (bool, error) {
	v, err := e.ifJSONPath(context.Background(), jsonPathDocument)
	if err != nil {
		return false, err
	}

	if v == nil {
		return false, nil
	} else if arr, ok := v.([]any); ok {
		return len(arr) > 0, nil
	} else if m, ok := v.(map[string]any); ok {
		return len(m) > 0, nil
	} else if b, ok := v.(bool); ok {
		return b, nil
	} else {
		return false, fmt.Errorf("unknown jsonpath result. Expected list, map or boolean. Got %#v", v)
	}
}

func (e *executeStep) check(jsonPathDocument map[string]any) error {
	return e.Check.check(jsonPathDocument)
}

func prepareMapFromRules(jsonPathDocument map[string]any, rules map[string]gval.Evaluable) (output map[string]any, err error) {
	output = map[string]any{}
	ctx := context.Background()
	for k, rule := range rules {
		v, err := rule(ctx, jsonPathDocument)
		if err != nil {
			return nil, fmt.Errorf("property %q: %w", k, err)
		}
		output[k] = v
	}
	return output, nil
}

func prepareMapFromSchemas(jsonPathDocument map[string]any, stepSchema *core.Schema, outerSchema *core.Schema, provided map[string]any) (output map[string]any, err error) {
	output = map[string]any{}
	for k, wantRef := range stepSchema.Properties {
		providesRef, ok := outerSchema.Properties[k]
		var v any
		if ok {
			v, ok = provided[k]
			if !ok {
				if wantRef.Value.Default != nil {
					v = wantRef.Value.Default
					ok = true
				} else if providesRef.Value.Default != nil {
					v = providesRef.Value.Default
					ok = true
				}
			}
		}
		if !ok {
			if slices.Contains(stepSchema.Required, k) {
				return nil, fmt.Errorf("required property %q missing in blueprint schema. Specify a manual JSON Path mapping", k)
			}
			continue
		}

		wantSchema := (*core.Schema)(wantRef.Value)
		providesSchema := (*core.Schema)(providesRef.Value)
		if !schemaPkg.CheckSimilarJsonSchemas(wantSchema, providesSchema) {
			return nil, fmt.Errorf("required property %q has different schemas. Specify a manual JSON Path mapping", k)
		}
		output[k] = v
	}

	return output, nil
}

func prepareMap(jsonPathDocument map[string]any, rules map[string]gval.Evaluable, stepSchema *core.Schema, outerSchema *core.Schema, provided map[string]any) (output map[string]any, err error) {
	if len(rules) > 0 {
		return prepareMapFromRules(jsonPathDocument, rules)
	} else {
		return prepareMapFromSchemas(jsonPathDocument, stepSchema, outerSchema, provided)
	}
}

func (e *executeStep) prepareParameters(jsonPathDocument map[string]any, outerSchema *core.Schema) (p core.Parameters, err error) {
	return prepareMap(jsonPathDocument, e.parametersJSONPath, e.executor.ParametersSchema(), outerSchema, jsonPathDocument["parameters"].(map[string]any))
}

func (e *executeStep) prepareConfigs(jsonPathDocument map[string]any, outerSchema *core.Schema) (c core.Configs, err error) {
	return prepareMap(jsonPathDocument, e.configsJSONPath, e.executor.ConfigsSchema(), outerSchema, jsonPathDocument["configs"].(map[string]any))
}
