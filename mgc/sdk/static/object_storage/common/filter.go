package common

import (
	"context"

	"magalu.cloud/core/pipeline"
)

type FilterParams struct {
	Include string `json:"include,omitempty" jsonschema:"description=Filename pattern to include"`
	Exclude string `json:"exclude,omitempty" jsonschema:"description=Filename pattern to exclude"`
}

func ApplyFilters(ctx context.Context, entries <-chan pipeline.WalkDirEntry, params FilterParams, cancel context.CancelCauseFunc) <-chan pipeline.WalkDirEntry {
	if params.Include != "" {
		includeFilter := pipeline.FilterRuleIncludeOnly[pipeline.WalkDirEntry]{
			Pattern: pipeline.FilterWalkDirEntryIncludeGlobMatch{Pattern: params.Include, CancelOnError: cancel},
		}

		entries = pipeline.Filter[pipeline.WalkDirEntry](ctx, entries, includeFilter)
	}

	if params.Exclude != "" {
		excludeFilter := pipeline.FilterRuleNot[pipeline.WalkDirEntry]{
			Not: pipeline.FilterWalkDirEntryIncludeGlobMatch{Pattern: params.Exclude, CancelOnError: cancel},
		}
		entries = pipeline.Filter[pipeline.WalkDirEntry](ctx, entries, excludeFilter)
	}

	return entries
}
