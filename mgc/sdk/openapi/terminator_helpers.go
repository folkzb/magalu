package openapi

import (
	"context"
	"fmt"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
	"go.uber.org/zap"
	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

func wrapInTerminatorExecutor(logger *zap.SugaredLogger, wtExt map[string]any, exec core.Executor) (core.TerminatorExecutor, error) {
	wt := &waitTermination{}
	if err := utils.DecodeValue(wtExt, wt); err != nil {
		logger.Warnw("error decoding extension wait-termination", "data", wtExt, "error", err)
	}

	if wt.MaxRetries <= 0 {
		wt.MaxRetries = defaultWaitTermination.MaxRetries
	}
	if wt.IntervalInSeconds <= 0 {
		wt.IntervalInSeconds = defaultWaitTermination.IntervalInSeconds
	}

	builder := gval.Full(jsonpath.PlaceholderExtension())
	jp, err := builder.NewEvaluable(wt.JSONPathQuery)
	if err == nil {
		tExec := core.NewTerminatorExecutorWithCheck(exec, wt.MaxRetries, wt.IntervalInSeconds, func(ctx context.Context, exec core.Executor, result core.ResultWithValue) (terminated bool, err error) {
			value := result.Value()
			v, err := jp(ctx, value)
			if err != nil {
				logger.Warnw("error evaluating jsonpath query", "query", wt.JSONPathQuery, "target", value, "error", err)
				return false, err
			}

			logger.Debugf("jsonpath expression %#v result is %#v", wt.JSONPathQuery, value)
			if v == nil {
				return false, nil
			} else if lst, ok := v.([]any); ok {
				return len(lst) > 0, nil
			} else if m, ok := v.(map[string]any); ok {
				return len(m) > 0, nil
			} else if b, ok := v.(bool); ok {
				return b, nil
			} else {
				logger.Warnw("unknown jsonpath result. Expected list, map or boolean", "result", value)
				return false, fmt.Errorf("unknown jsonpath result. Expected list, map or boolean. Got %+v", value)
			}
		})
		return tExec, nil
	} else {
		logger.Warnw("error parsing jsonpath. Executing without polling", "expression", wt.JSONPathQuery, "error", err)
		return nil, err
	}
}
