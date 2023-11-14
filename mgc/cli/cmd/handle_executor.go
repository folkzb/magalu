package cmd

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"magalu.cloud/cli/ui"
	"magalu.cloud/core"
	"magalu.cloud/core/progress_report"
	mgcSdk "magalu.cloud/sdk"
)

func handleExecutorResult(ctx context.Context, sdk *mgcSdk.Sdk, cmd *cobra.Command, result core.Result, err error) error {
	if err != nil {
		var failedTerminationError core.FailedTerminationError
		if errors.As(err, &failedTerminationError) {
			_ = formatResult(sdk, cmd, failedTerminationError.Result)
		}
		return err
	}

	return formatResult(sdk, cmd, result)
}

func handleExecutor(
	ctx context.Context,
	sdk *mgcSdk.Sdk,
	cmd *cobra.Command,
	exec core.Executor,
	parameters core.Parameters,
	configs core.Configs,
) (core.Result, error) {
	if pb != nil {
		ctx = progress_report.NewContext(ctx, pb.ReportProgress)
	}

	if cExec, ok := core.ExecutorAs[core.ConfirmableExecutor](exec); ok && !getBypassConfirmationFlag(cmd) {
		msg := cExec.ConfirmPrompt(parameters, configs)
		run, err := ui.Confirm(msg)
		if err != nil {
			return nil, err
		}

		if !run {
			return nil, core.UserDeniedConfirmationError{Prompt: msg}
		}
	}

	if t := getTimeoutFlag(cmd); t > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, t)
		defer cancel()
	}

	waitTermination := getWaitTerminationFlag(cmd)
	var cb core.RetryUntilCb
	if tExec, ok := core.ExecutorAs[core.TerminatorExecutor](exec); ok && waitTermination {
		cb = func() (result core.Result, err error) {
			return tExec.ExecuteUntilTermination(ctx, parameters, configs)
		}
	} else {
		cb = func() (result core.Result, err error) {
			return exec.Execute(ctx, parameters, configs)
		}
	}

	retry, err := getRetryUntilFlag(cmd)
	if err != nil {
		return nil, err
	}

	result, err := retry.Run(ctx, cb)

	err = handleExecutorResult(ctx, sdk, cmd, result, err)
	if err != nil {
		return nil, err
	}

	return result, err
}
