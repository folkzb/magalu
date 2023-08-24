package auth

import (
	"go.uber.org/zap"
	"magalu.cloud/core/logger"
)

type pkgSymbol struct{}

var pkgLogger *zap.SugaredLogger

func initPkgLogger() *zap.SugaredLogger {
	if pkgLogger == nil {
		pkgLogger = logger.New[pkgSymbol]()
	}
	return pkgLogger
}
