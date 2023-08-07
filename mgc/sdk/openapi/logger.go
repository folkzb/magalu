package openapi

import (
	"go.uber.org/zap"
	"magalu.cloud/core"
)

type pkgSymbol struct{}

var pkgLogger *zap.SugaredLogger

func initPkgLogger() *zap.SugaredLogger {
	if pkgLogger == nil {
		pkgLogger = core.NewLogger[pkgSymbol]()
	}
	return pkgLogger
}
