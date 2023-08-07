package core

import (
	"go.uber.org/zap"
)

type pkgSymbol struct{}

var pkgLogger *zap.SugaredLogger

func initPkgLogger() *zap.SugaredLogger {
	if pkgLogger == nil {
		return NewLogger[pkgSymbol]()
	}
	return pkgLogger
}
