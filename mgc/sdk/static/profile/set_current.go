package profile

import (
	"context"
	"errors"

	"magalu.cloud/core"
	"magalu.cloud/core/profile_manager"
	"magalu.cloud/core/utils"
)

type setCurrentParams struct {
	Name string `json:"name" jsonschema_description:"Profile name"`
}

var getSetCurrent = utils.NewLazyLoader[core.Executor](func() core.Executor {
	exec := core.NewStaticExecute(
		core.DescriptorSpec{
			Name:        "set-current",
			Description: "Sets profile to be used",
		},
		setCurrent,
	)

	return core.NewExecuteResultOutputOptions(exec, func(exec core.Executor, result core.Result) string {
		return "template={{.Name}}\n"
	})
})

func setCurrent(ctx context.Context, params setCurrentParams, _ struct{}) (*profile_manager.Profile, error) {
	m := profile_manager.FromContext(ctx)
	if m == nil {
		return nil, ProfileError{Name: "", Err: errors.New("Couldn't get ProfileManager from context")}
	}

	p, err := m.Get(params.Name)
	if err != nil {
		return nil, ProfileError{Name: params.Name, Err: err}
	}

	err = m.SetCurrent(p)
	if err != nil {
		return nil, ProfileError{Name: params.Name, Err: err}
	}

	return p, nil
}
