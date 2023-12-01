package schema_flags

import (
	mgcLoggerPkg "magalu.cloud/core/logger"
)

var logger = mgcLoggerPkg.NewLazy[SchemaFlagValueDesc]()
