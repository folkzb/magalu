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
		initLoggerFromConfig(nil)
	}
	return rootLogger
}

func initLoggerFromConfig(config *zap.Config) {
	// TODO: set Development/Production config based on env var
	var logger *zap.Logger
	if config == nil {
		logger = zap.Must(zap.NewDevelopment())
	} else {
		logger = zap.Must(config.Build())
	}

	rootLogger = logger.Sugar().Named("mgc")
}

func InitLoggerFilter(opts ...zap.Option) {
	rootLogger = RootLogger().WithOptions(opts...)
}
