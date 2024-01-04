package tenant

import (
	"context"
	"fmt"

	"magalu.cloud/core"
	mgcAuthPkg "magalu.cloud/core/auth"
	"magalu.cloud/core/utils"
)

type tenantCurrentResult struct {
	ID string
}

var getCurrent = utils.NewLazyLoader[core.Executor](newCurrent)

func newCurrent() core.Executor {
	return core.NewStaticExecuteSimple(
		core.DescriptorSpec{
			Name:        "current",
			Summary:     "Get the currently active Tenant",
			Description: `The current Tenant is used for all Magalu HTTP requests`,
		},
		func(ctx context.Context) (*tenantCurrentResult, error) {
			auth := mgcAuthPkg.FromContext(ctx)
			if auth == nil {
				return nil, fmt.Errorf("unable to get auth from context")
			}

			id, err := auth.CurrentTenantID()
			if err != nil {
				return nil, err
			}

			if id == "" {
				return nil, fmt.Errorf("current tenant ID is empty. Try logging in at least once or run the 'auth tenant select' operation")
			}

			return &tenantCurrentResult{ID: id}, nil
		},
	)
}
