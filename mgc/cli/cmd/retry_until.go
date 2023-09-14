package cmd

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/PaesslerAG/gval"
	"github.com/PaesslerAG/jsonpath"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
	"magalu.cloud/core"
)

const (
	retryUntilFlag                string = "cli.retry-until"
	retryUntilFlagFormat          string = "\"retries,interval,condition\""
	retryUntilFlagConditionFormat string = "\"engine=value\""
)

var retryUntilTemplateStrings = []string{
	"finished",
	"terminated",
	"true",
}

type retryUntilCheck func(ctx context.Context, value core.Value) (finished bool, err error)
type retryUntilSpec struct {
	maxRetries int
	interval   time.Duration
	condition  string
	check      retryUntilCheck
}

type retryUntilCb func() (result core.Result, err error)

func (r *retryUntilSpec) run(ctx context.Context, cb retryUntilCb) (result core.Result, err error) {
	if r == nil {
		return cb()
	}

	for i := 0; i < r.maxRetries; i++ {
		result, err = cb()
		if err != nil {
			return result, err
		}
		resultWithValue, ok := core.ResultAs[core.ResultWithValue](result)
		if !ok {
			return result, fmt.Errorf("result has no value")
		}
		finished, err := r.check(ctx, resultWithValue.Value())
		if err != nil {
			return result, err
		}
		if finished {
			return result, nil
		}

		timer := time.NewTimer(r.interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}

	msg := fmt.Sprintf("condition %q exceeded maximum retries %d with interval %s", r.condition, r.maxRetries, r.interval)
	return nil, core.FailedTerminationError{Result: result, Message: msg}
}

func addRetryUntilFlag(cmd *cobra.Command) {
	cmd.Root().PersistentFlags().StringP(
		retryUntilFlag,
		"U",
		"",
		"Retry the action with the same parameters until the given condition is met. The flag parameters use the format: "+retryUntilFlagFormat+", where \"retries\" is a positive integer, \"interval\" is a duration (ex: 2s) and \"condition\" is a "+retryUntilFlagConditionFormat+" pair such as \"jsonpath=expression\"",
	)
}

func getRetryUntilFlag(cmd *cobra.Command) (spec *retryUntilSpec, err error) {
	var v string
	v, err = cmd.Root().PersistentFlags().GetString(retryUntilFlag)
	if err != nil {
		return
	}

	return parseRetryUntilFlag(v)
}

func parseRetryUntilFlag(v string) (spec *retryUntilSpec, err error) {
	if v == "" {
		return nil, nil
	}

	p := strings.SplitN(v, ",", 3)
	if len(p) != 3 {
		err = fmt.Errorf("--%s value must be in the format %s", retryUntilFlag, retryUntilFlagFormat)
		return
	}

	spec = &retryUntilSpec{}

	if _, err = fmt.Sscanf(p[0], "%d", &spec.maxRetries); err != nil {
		err = fmt.Errorf("--%s: failed to parse retries: %w", retryUntilFlag, err)
		return
	}

	spec.interval, err = time.ParseDuration(p[1])
	if err != nil {
		err = fmt.Errorf("--%s: failed to parse interval: %w", retryUntilFlag, err)
		return
	}

	spec.condition = p[2]
	p = strings.SplitN(spec.condition, "=", 2)
	if len(p) != 2 {
		err = fmt.Errorf("--%s condition must be in the format %s", retryUntilFlag, retryUntilFlagConditionFormat)
		return
	}
	switch p[0] {
	default:
		err = fmt.Errorf("--%s unknown condition engine: %s, supported: jsonpath|template", retryUntilFlag, p[0])
		return

	case "jsonpath":
		spec.check, err = parseRetryUntilJsonPath(p[1])
	case "template":
		spec.check, err = parseRetryUntilTemplate(p[1])
	}
	return
}

func parseRetryUntilJsonPath(expression string) (check retryUntilCheck, err error) {
	builder := gval.Full(jsonpath.PlaceholderExtension())
	jp, err := builder.NewEvaluable(expression)
	if err != nil {
		return nil, err
	}

	check = func(ctx context.Context, value core.Value) (finished bool, err error) {
		v, err := jp(ctx, value)
		if err != nil {
			return false, err
		}

		if v == nil {
			return false, nil
		} else if lst, ok := v.([]any); ok {
			return len(lst) > 0, nil
		} else if m, ok := v.(map[string]any); ok {
			return len(m) > 0, nil
		} else if b, ok := v.(bool); ok {
			return b, nil
		} else {
			return false, fmt.Errorf("unknown jsonpath result. Expected list, map or boolean. Got %+v", value)
		}
	}

	return
}

func parseRetryUntilTemplate(expression string) (check retryUntilCheck, err error) {
	tmpl, err := template.New(retryUntilFlag).Parse(expression)
	if err != nil {
		return nil, err
	}

	check = func(ctx context.Context, value core.Value) (finished bool, err error) {
		var buf bytes.Buffer
		err = tmpl.Execute(&buf, value)
		if err != nil {
			return false, err
		}
		s := buf.String()
		s = strings.Trim(s, " \t\n\r")
		return slices.Contains(retryUntilTemplateStrings, s), nil
	}

	return
}
