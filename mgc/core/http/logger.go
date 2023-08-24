package http

import (
	"go.uber.org/zap"
	logger1 "magalu.cloud/core/logger"
)

type pkgSymbol struct{}

var pkgLogger *zap.SugaredLogger

func initPkgLogger() *zap.SugaredLogger {
	if pkgLogger == nil {
		pkgLogger = logger1.New[pkgSymbol]()
	}
	return pkgLogger
}
