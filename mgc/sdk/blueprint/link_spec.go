package blueprint

import (
	"errors"
	"fmt"

	"github.com/PaesslerAG/gval"
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

type linkSpec struct {
	name string // set externally

	Description     string                      `json:"description"`
	IsInternal      bool                        `json:"hidden"`
	Target          core.RefPath                `json:"target"`
	WaitTermination *core.WaitTerminationConfig `json:"waitTermination,omitempty"`
	Parameters      map[string]string           `json:"parameters,omitempty"`
	Configs         map[string]string           `json:"configs,omitempty"`

	// Materialized/Parsed values based on the JSON fields. They are populated in validate():

	executor           core.Executor
	parametersJSONPath map[string]gval.Evaluable
	configsJSONPath    map[string]gval.Evaluable
}

func (e *linkSpec) validate() (err error) {
	if e.Description == "" {
		return errors.New("missing description")
	}

	if e.Target == "" {
		return errors.New("missing target")
	}

	if e.parametersJSONPath == nil {
		m := make(map[string]gval.Evaluable)
		for k, v := range e.Parameters {
			eval, err := utils.NewJsonPath(v)
			if err != nil {
				return fmt.Errorf("parameters[%q]: %w", k, err)
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
				return fmt.Errorf("configs[%q]: %w", k, err)
			}
			m[k] = eval
		}
		e.configsJSONPath = m
	}

	return nil
}

func (e *linkSpec) resolve(refResolver *core.BoundRefPathResolver) (err error) {
	e.executor, err = core.ResolveExecutorPath(refResolver, e.Target)
	return
}
