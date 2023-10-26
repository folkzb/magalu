package progress_bar

import (
	"errors"
	"fmt"
	"io"
	"sync"

	"github.com/jedib0t/go-pretty/v6/progress"
	"magalu.cloud/core/progress_report"
)

func parseUnits(units progress_report.Units) progress.Units {
	switch units {
	case progress_report.UnitsNone:
		return progress.UnitsDefault
	case progress_report.UnitsBytes:
		return progress.UnitsBytes
	default:
		return progress.UnitsDefault
	}
}

type ProgressBar struct {
	progress.Writer
	trackers sync.Map
}

func New() *ProgressBar {
	writer := progress.NewWriter()
	writer.SetAutoStop(true)
	writer.SetUpdateFrequency(progress.DefaultUpdateFrequency)
	writer.SetMessageWidth(30)
	writer.SetTrackerPosition(progress.PositionRight)
	writer.SetTrackerLength(progress.DefaultLengthTracker)
	go writer.Render()
	return &ProgressBar{
		Writer: writer,
	}
}

func (pb *ProgressBar) ReportProgress(msg string, done, total uint64, units progress_report.Units, reportErr error) {
	tracker, found := pb.trackers.LoadOrStore(msg,
		&progress.Tracker{
			Message: msg,
			Total:   int64(total),
			Units:   parseUnits(units),
		},
	)
	castTracker := tracker.(*progress.Tracker)
	if !found {
		pb.Writer.AppendTracker(castTracker)
	}

	castTracker.SetValue(int64(done))
	if reportErr != nil && !errors.Is(reportErr, io.EOF) {
		if errors.Is(reportErr, progress_report.ErrorProgressDone) {
			castTracker.MarkAsDone()
			return
		}
		castTracker.MarkAsErrored()
		castTracker.UpdateMessage(fmt.Sprintf("%s [%s]", msg, reportErr.Error()))
	}
}
