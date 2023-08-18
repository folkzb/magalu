package core

import (
	"reflect"

	"go.uber.org/zap"
)

var rootLogger *zap.SugaredLogger

func NewLogger[T any]() *zap.SugaredLogger {
	return RootLogger().Named(reflect.TypeOf(new(T)).Elem().PkgPath())
}

func RootLogger() *zap.SugaredLogger {
	if rootLogger == nil {
		rootLogger = zap.Must(zap.NewProduction()).Sugar()
	}
	return rootLogger
}

func SetRootLogger(logger *zap.SugaredLogger) {
	rootLogger = logger
}
