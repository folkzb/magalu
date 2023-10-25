package progress_report

import (
	"context"
	"errors"
)

type contextKey string

const progressReporterKey contextKey = "magalu.cli/core/progressreport"

type Units int

const (
	UnitsNone = iota
	UnitsBytes
)

// Not a real error, just a flag that can be used to tell progress reporter that
// this process is Done
var ErrorProgressDone = errors.New("Progress is done")

// Progress reporting function signature
type ReportProgress func(msg string, done, total uint64, units Units, reportErr error)

// Insert a progress reporting function in the context
//
// This serves kind of like an interface, as the implementation of what to do
// when progress is reported is done by the caller
func NewContext(ctx context.Context, updateFunc ReportProgress) context.Context {
	return context.WithValue(ctx, progressReporterKey, updateFunc)
}

// Retrieves the progress reporting function from the context
func FromContext(ctx context.Context) ReportProgress {
	pr, ok := ctx.Value(progressReporterKey).(ReportProgress)
	if !ok {
		return dummyProgressReport
	}
	return pr
}

// Function does nothing, only exists to not return nil values
func dummyProgressReport(msg string, done, total uint64, units Units, err error) {
}
