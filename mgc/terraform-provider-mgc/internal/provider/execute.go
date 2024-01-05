package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"magalu.cloud/core"
)

func validateResult(d *diag.Diagnostics, result core.ResultWithValue) error {
	err := result.ValidateSchema()
	if err != nil {
		d.AddWarning(
			"Operation output mismatch",
			fmt.Sprintf("Result has invalid structure: %v", err),
		)
	}
	return err
}

// Does not return error, check for 'diag.HasError' to see if operation was successful
func execute(
	resName tfName,
	ctx context.Context,
	exec core.Executor,
	params core.Parameters,
	configs core.Configs,
	diag *diag.Diagnostics,
) core.ResultWithValue {
	var result core.Result
	var err error

	tflog.Debug(ctx, fmt.Sprintf("[resource] will %s new %s resource - request info with params: %#v and configs: %#v", exec.Name(), resName, params, configs))
	if tExec, ok := core.ExecutorAs[core.TerminatorExecutor](exec); ok {
		tflog.Debug(ctx, "[resource] running as TerminatorExecutor")
		result, err = tExec.ExecuteUntilTermination(ctx, params, configs)
	} else {
		tflog.Debug(ctx, "[resource] running as Executor")
		result, err = exec.Execute(ctx, params, configs)
	}
	if err != nil {
		diag.AddError(
			fmt.Sprintf("Unable to %s %s", exec.Name(), resName),
			fmt.Sprintf("Service returned with error: %v", err),
		)
		return nil
	}

	resultWithValue, ok := core.ResultAs[core.ResultWithValue](result)
	if !ok {
		diag.AddError(
			"Operation output mismatch",
			fmt.Sprintf("result has no value %#v", result),
		)
		return nil
	}

	/* TODO:
	if err := validateResult(diag, result); err != nil {
		return
	}
	*/
	_ = validateResult(diag, resultWithValue) // just ignore errors for now

	return resultWithValue
}
