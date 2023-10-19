package blueprint

import (
	"context"
	"encoding/json"
	"time"

	"magalu.cloud/core"
	"magalu.cloud/core/utils"
)

type retryUntil struct {
	conditionSpec `json:",squash"` // nolint

	MaxRetries int           `json:"maxRetries,omitempty"`
	Interval   time.Duration `json:"interval,omitempty"`
}

func (r *retryUntil) UnmarshalJSON(data []byte) (err error) {
	m := map[string]any{}
	err = json.Unmarshal(data, &m) // decoding interval to time.Duration would fail
	if err != nil {
		return
	}
	return utils.DecodeValue(m, r) // nicely decodes time.Duration
}

var _ json.Unmarshaler = (*retryUntil)(nil)

func (r *retryUntil) create(getDocumentForValue func(value core.Value) map[string]any) *core.RetryUntil {
	if r == nil {
		return nil
	}

	return &core.RetryUntil{
		MaxRetries: r.MaxRetries,
		Interval:   r.Interval,
		Check: func(ctx context.Context, value core.Value) (finished bool, err error) {
			doc := getDocumentForValue(value)
			return r.check(doc)
		},
	}
}

func (r *retryUntil) run(ctx context.Context, cb core.RetryUntilCb, getDocumentForValue func(value core.Value) map[string]any) (result core.Result, err error) {
	// nil pointer retryUntil and core.RetryUntil are ok
	return r.create(getDocumentForValue).Run(ctx, cb)
}

func (r *retryUntil) validate() (err error) {
	if r == nil {
		return nil
	}

	err = r.conditionSpec.validate()
	if err != nil {
		return err
	}

	return nil
}
