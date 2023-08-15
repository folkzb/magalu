package cmd

import (
	"go.uber.org/zap"
	"magalu.cloud/core"
)

type pkgSymbol struct{}

var loggerInstance *zap.SugaredLogger

func logger() *zap.SugaredLogger {
	if loggerInstance == nil {
		loggerInstance = core.NewLogger[pkgSymbol]()
	}
	return loggerInstance
}
