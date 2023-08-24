package logger

import (
	"fmt"
	"reflect"
	"time"

	"magalu.cloud/core"

	"github.com/invopop/jsonschema"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var rootLogger *zap.SugaredLogger

func New[T any]() *zap.SugaredLogger {
	return Root().Named(reflect.TypeOf(new(T)).Elem().PkgPath())
}

func Root() *zap.SugaredLogger {
	if rootLogger == nil {
		rootLogger = zap.Must(zap.NewProduction()).Sugar()
	}
	return rootLogger
}

func SetRoot(logger *zap.SugaredLogger) {
	rootLogger = logger
}

func ConfigSchema() (*core.Schema, error) {
	reflector := jsonschema.Reflector{DoNotReference: true, Mapper: zapMapper}
	s, err := core.ToCoreSchema(reflector.Reflect(zap.Config{}))
	if err != nil {
		return nil, fmt.Errorf("unable to create JSON Schema for type '%T': %w", zap.Config{}, err)
	}

	s.Description = "Logger configuration. For more information see https://pkg.go.dev/go.uber.org/zap#Config"

	return s, nil
}

func levelEncoder(zapcore.Level, zapcore.PrimitiveArrayEncoder)        {}
func timeEncoder(time.Time, zapcore.PrimitiveArrayEncoder)             {}
func durationEncoder(time.Duration, zapcore.PrimitiveArrayEncoder)     {}
func callerEncoder(zapcore.EntryCaller, zapcore.PrimitiveArrayEncoder) {}
func nameEncoder(string, zapcore.PrimitiveArrayEncoder)                {}

// The zapMapper function is necessary because some zapcore.EncoderConfig
// fields are functions, and it's not possiple to reflect these fields
// to a jsonschema.Schema. So we have to tell jsonschema how to handle them

func zapMapper(t reflect.Type) *jsonschema.Schema {
	switch t {
	case reflect.TypeOf(zap.AtomicLevel{}):
		return &jsonschema.Schema{
			Type: "string",
			Enum: []any{
				"debug",
				"info",
				"warn",
				"error",
				"panic",
				"dpanic",
				"fatal",
			},
		}
	case reflect.TypeOf(zapcore.LevelEncoder(levelEncoder)):
		return &jsonschema.Schema{
			Type: "string",
			Enum: []any{
				"capital",
				"color",
				"capitalColor",
				"default",
			}}
	case reflect.TypeOf(zapcore.TimeEncoder(timeEncoder)):
		return &jsonschema.Schema{
			Type: "string",
			Enum: []any{
				"rfc3339nano", "RFC3339Nano",
				"rfc3339", "RFC3339",
				"iso8601", "ISO8601",
				"millis",
				"nanos",
				"default",
			}}
	case reflect.TypeOf(zapcore.DurationEncoder(durationEncoder)):
		return &jsonschema.Schema{
			Type: "string",
			Enum: []any{
				"string",
				"nanos",
				"ms",
				"default",
			},
		}
	case reflect.TypeOf(zapcore.CallerEncoder(callerEncoder)):
		return &jsonschema.Schema{
			Type: "string",
			Enum: []any{
				"string",
				"default",
			},
		}
	case reflect.TypeOf(zapcore.NameEncoder(nameEncoder)):
		return &jsonschema.Schema{
			Type: "string",
			Enum: []any{
				"full",
				"default",
			},
		}
	default:
		return nil
	}
}
