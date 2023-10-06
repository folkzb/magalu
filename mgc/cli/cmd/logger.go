package cmd

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	coreLogger "magalu.cloud/core/logger"
	mgcSdk "magalu.cloud/sdk"
	"moul.io/zapfilter"
)

type pkgSymbol struct{}

var loggerInstance *zap.SugaredLogger

func logger() *zap.SugaredLogger {
	if loggerInstance == nil {
		loggerInstance = coreLogger.New[pkgSymbol]()
	}
	return loggerInstance
}

func newLogConfig() zap.Config {
	zapConfig := zap.NewProductionConfig()
	zapConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)             // it's widely used, zapfilter will default to "warn+:*"
	zapConfig.Encoding = "console"                                     // after all, it's a CLI
	zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder  // INFO, DEBUG...
	zapConfig.EncoderConfig.EncodeTime = zapcore.EpochNanosTimeEncoder // smaller yet high-resolution
	zapConfig.EncoderConfig.CallerKey = ""                             // do not show file:line
	zapConfig.EncoderConfig.TimeKey = ""                               // do not show timestamp by default
	return zapConfig
}

func initLogger(sdk *mgcSdk.Sdk, filterRules string) error {
	zapConfig := newLogConfig()

	if err := sdk.Config().Get(loggerConfigKey, &zapConfig); err != nil {
		return fmt.Errorf("unable to get logger configuration: %w", err)
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return fmt.Errorf("unable to build logger. Make sure a valid configuration was provided: %w", err)
	}

	filterOpt := zap.WrapCore(func(c zapcore.Core) zapcore.Core {
		return zapfilter.NewFilteringCore(c, zapfilter.MustParseRules(filterRules))
	})

	logger = logger.WithOptions(filterOpt)
	coreLogger.SetRoot(logger.Sugar())

	return nil
}
