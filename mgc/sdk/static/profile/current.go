package profile

import (
	"context"
	"errors"

	"magalu.cloud/core"
	"magalu.cloud/core/profile_manager"
	"magalu.cloud/core/utils"
)

var getCurrent = utils.NewLazyLoader[core.Executor](func() core.Executor {
	exec := core.NewStaticExecuteSimple(
		core.DescriptorSpec{
			Name:        "current",
			Description: "Shows current selected profile. Any changes to auth or config values will only affect this profile",
		},
		current,
	)

	return core.NewExecuteResultOutputOptions(exec, func(exec core.Executor, result core.Result) string {
		return "template={{.Name}}\n"
	})
})

func current(ctx context.Context) (*profile_manager.Profile, error) {
	m := profile_manager.FromContext(ctx)
	if m == nil {
		return nil, ProfileError{Name: "", Err: errors.New("Couldn't get ProfileManager from context")}
	}

	return m.Current(), nil
}
