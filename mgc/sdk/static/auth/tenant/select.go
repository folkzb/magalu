package tenant

import (
	"context"
	"fmt"

	"magalu.cloud/core"
)

type tenantSetParams struct {
	ID string `jsonschema_description:"The UUID of the desired Tenant. To list all possible IDs, run auth tenant list"`
}

func newSelect() core.Executor {
	executor := core.NewStaticExecute(
		"select",
		"",
		"Set the active Tenant to be used for all subsequential requests",
		selectTenant,
	)

	return core.NewExecuteResultOutputOptions(executor, func(exec core.Executor, result core.Value) string {
		return "template=Success! Current tenant changed to {{.id}}\n"
	})
}

func selectTenant(ctx context.Context, params tenantSetParams, _ struct{}) (*core.TenantAuth, error) {
	auth := core.AuthFromContext(ctx)
	if auth == nil {
		return nil, fmt.Errorf("Unable to get auth from context")
	}
	return auth.SelectTenant(params.ID)
}
