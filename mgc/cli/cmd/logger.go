package cmd

import (
	"go.uber.org/zap"
	coreLogger "magalu.cloud/core/logger"
)

type pkgSymbol struct{}

var loggerInstance *zap.SugaredLogger

func logger() *zap.SugaredLogger {
	if loggerInstance == nil {
		loggerInstance = coreLogger.New[pkgSymbol]()
	}
	return loggerInstance
}
