package tenant

import (
	"context"
	"fmt"

	"magalu.cloud/core"
	"magalu.cloud/core/auth"
)

func newList() core.Executor {
	return core.NewStaticExecuteSimple(
		"list",
		"",
		"List all available tenants for current login",
		ListTenants,
	)
}

func ListTenants(ctx context.Context) ([]*auth.Tenant, error) {
	auth := auth.FromContext(ctx)
	if auth == nil {
		return nil, fmt.Errorf("unable to get auth from context")
	}
	return auth.ListTenants()
}
