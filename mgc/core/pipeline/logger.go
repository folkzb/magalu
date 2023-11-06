package pipeline

import (
	"context"

	"go.uber.org/zap"
	mgcLoggerPkg "magalu.cloud/core/logger"
)

var logger = mgcLoggerPkg.NewLazy[ProcessStatus]()

type contextLoggerKey string

var ctxWrappedKey contextLoggerKey = "magalu.cloud/core/pipeline"

// Get the logger from context or return the default logger.
//
// Loggers are used from context in order to avoid passing it explicitly everywhere,
// which would make APIs cumbersome to use.
func FromContext(ctx context.Context) *zap.SugaredLogger {
	if v := ctx.Value(ctxWrappedKey); v != nil {
		if l, ok := v.(*zap.SugaredLogger); ok {
			return l
		}
	}
	return logger()
}

// Create a new context with a new logger.
//
// Any existing loggers will be superseded by the given one in the returned context and others.
// Parent context logger (if exists) is untouched.
func NewContext(parentCtx context.Context, logger *zap.SugaredLogger) context.Context {
	return context.WithValue(parentCtx, ctxWrappedKey, logger)
}
