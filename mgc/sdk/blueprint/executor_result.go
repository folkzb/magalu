package blueprint

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"go.uber.org/zap"
	"golang.org/x/exp/maps"
	"magalu.cloud/core"
	schemaPkg "magalu.cloud/core/schema"
	"magalu.cloud/core/utils"
)

type executorStepResult struct {
	step       *executeStep
	parameters core.Parameters
	configs    core.Configs
	result     core.Result
	err        error
	skipped    bool
}

type executorResult struct {
	core.ResultSource
	steps          []*executorStepResult
	logger         *zap.SugaredLogger
	resultJsonPath string
	value          core.Value

	// these are populated by jsonPathDocument:

	lastJsonPathDocument map[string]any
	lastJsonPathStep     int
}

func (r *executorResult) Source() core.ResultSource {
	return r.ResultSource
}

func (r *executorResult) Schema() *core.Schema {
	return r.Executor.ResultSchema()
}

func (r *executorResult) ValidateSchema() error {
	return r.Schema().VisitJSON(r.Value(), openapi3.MultiErrors())
}

func (r *executorResult) Value() core.Value {
	return r.value
}

func (r *executorResult) adjustValueToSchema(value core.Value) (v core.Value, err error) {
	return adjustValueToSchema(r.Schema(), value, r.logger.Named("result"))
}

// Try our best to be compliant to the desired schema, recursive
func adjustValueToSchema(schema *schemaPkg.Schema, value core.Value, logger *zap.SugaredLogger) (v core.Value, err error) {
	switch schema.Type {
	case "boolean":
		var decoded *bool
		if decoded, err = utils.DecodeNewValue[bool](v); err != nil {
			return
		} else {
			return *decoded, nil
		}

	case "integer":
		var decoded *int64
		if decoded, err = utils.DecodeNewValue[int64](v); err != nil {
			return
		} else {
			return *decoded, nil
		}

	case "number":
		var decoded *float64
		if decoded, err = utils.DecodeNewValue[float64](v); err != nil {
			return
		} else {
			return *decoded, nil
		}

	case "string":
		var decoded *string
		if decoded, err = utils.DecodeNewValue[string](v); err != nil {
			return
		} else {
			return *decoded, nil
		}

	case "array":
		var decoded *[]any
		if decoded, err = utils.DecodeNewValue[[]any](v); err != nil {
			return
		} else {
			sl := *decoded
			itemSchema := (*schemaPkg.Schema)(schema.Items.Value)
			if itemSchema != nil {
				for i, e := range sl {
					sl[i], err = adjustValueToSchema(itemSchema, e, logger.Named(fmt.Sprint(i)))
					if err != nil {
						return nil, fmt.Errorf("item %d: %w", i, err)
					}
				}
			}
			return sl, nil
		}

	case "object":
		var decoded *map[string]any
		if decoded, err = utils.DecodeNewValue[map[string]any](v); err != nil {
			return
		} else {
			m := map[string]any{} // new map so we skip unknown keys
			for k, e := range *decoded {
				propRef := schema.Properties[k]
				if propRef == nil {
					if schema.AdditionalProperties.Has == nil || !*schema.AdditionalProperties.Has {
						continue
					}
				} else {
					propSchema := (*schemaPkg.Schema)(propRef.Value)
					if propSchema != nil {
						e, err = adjustValueToSchema(propSchema, e, logger.Named(k))
						if err != nil {
							return nil, fmt.Errorf("property %q: %w", k, err)
						}
					}
				}
				m[k] = e
			}
			return m, nil
		}

	default:
		logger.Warnw("unhandled schema type", "jsonType", schema.Type, "schema", schema, "value", value)
		return value, nil
	}
}

func (r *executorResult) realizeValue() (err error) {
	jsonPathDocument := r.jsonPathDocument()
	logger := r.logger.With("jsonPathDocument", jsonPathDocument)
	if r.resultJsonPath == "" {
		r.value = nil
		if last, ok := jsonPathDocument["last"].(map[string]any); !ok {
			logger.Warnw("all steps were skipped")
		} else if value, ok := last["result"]; !ok {
			logger.Warnw("last step result has no value")
		} else {
			r.value, err = r.adjustValueToSchema(value)
		}
	} else {
		r.value, err = utils.GetJsonPath(r.resultJsonPath, jsonPathDocument)
	}

	if err != nil {
		logger.Warnw(
			"could not create result",
			"resultJsonPath", r.resultJsonPath,
			"error", err,
		)
		return fmt.Errorf("could not create result: %w", err)
	}

	return nil
}

func (r *executorResult) finalize() (result core.ResultWithValue, err error) {
	for i := len(r.steps) - 1; i >= 0; i-- {
		step := r.steps[i]
		if step.skipped {
			continue
		}
		if step.err != nil {
			return nil, fmt.Errorf("step %q finished with error: %w", step.step.Id, step.err)
		}
		break
	}

	err = r.realizeValue()
	return r, err
}

func (r *executorResult) reportResult(step *executeStep, parameters core.Parameters, configs core.Configs, result core.Result) {
	r.steps = append(r.steps, &executorStepResult{step, parameters, configs, result, nil, false})
}

func (r *executorResult) reportError(step *executeStep, parameters core.Parameters, configs core.Configs, err error) {
	r.steps = append(r.steps, &executorStepResult{step, parameters, configs, nil, err, false})
}

func (r *executorResult) skip(step *executeStep) {
	r.steps = append(r.steps, &executorStepResult{step, nil, nil, nil, nil, true})
}

func getResultValueJsonPathDocument(result core.Result) any {
	if rVal, ok := core.ResultAs[core.ResultWithValue](result); ok {
		return rVal.Value()
	}
	return nil
}

func (r *executorResult) initJsonPathDocument() {
	if r.lastJsonPathDocument == nil {
		r.lastJsonPathDocument = map[string]any{
			"parameters": r.ResultSource.Parameters,
			"configs":    r.ResultSource.Configs,
			"steps":      map[string]any{},
			"last":       nil,
		}
		r.lastJsonPathStep = -1
	}
}

func createStepResultJsonDocument(stepResult *executorStepResult) map[string]any {
	return map[string]any{
		"id":         stepResult.step.Id,
		"parameters": stepResult.parameters,
		"configs":    stepResult.configs,
		"result":     getResultValueJsonPathDocument(stepResult.result),
		"error":      stepResult.err,
		"skipped":    stepResult.skipped,
	}
}

func (r *executorResult) fillMissingSteps() {
	nSteps := len(r.steps)
	start := r.lastJsonPathStep + 1
	if start == nSteps {
		return
	}

	steps := r.lastJsonPathDocument["steps"].(map[string]any)
	var lastProcessedResultJsonDocument map[string]any
	for i := start; i < nSteps; i++ {
		stepResult := r.steps[i]
		resultJsonDocument := createStepResultJsonDocument(stepResult)
		steps[stepResult.step.Id] = resultJsonDocument
		if !stepResult.skipped {
			lastProcessedResultJsonDocument = resultJsonDocument
		}
	}
	r.lastJsonPathStep = nSteps - 1
	if lastProcessedResultJsonDocument != nil {
		r.lastJsonPathDocument["last"] = lastProcessedResultJsonDocument
	}
}

func (r *executorResult) jsonPathDocument() (doc map[string]any) {
	r.initJsonPathDocument()
	r.fillMissingSteps()
	return r.lastJsonPathDocument
}

func (r *executorResult) jsonPathDocumentWithResult() map[string]any {
	doc := maps.Clone(r.jsonPathDocument())
	doc["result"] = r.Value()
	return doc
}

func (r *executorResult) jsonPathDocumentWithCurrent(step *executeStep, parameters core.Parameters, configs core.Configs, value core.Value) map[string]any {
	doc := maps.Clone(r.jsonPathDocument())
	// mimic the final json document, gives checkers the full context of other steps
	current := map[string]any{
		"id":         step.Id,
		"parameters": parameters,
		"configs":    configs,
		"result":     value,
	}
	current["result"] = value
	doc["current"] = current
	steps := maps.Clone(doc["steps"].(map[string]any))
	steps[step.Id] = current
	return doc
}

var _ core.ResultWithValue = (*executorResult)(nil)