package schema

import (
	mgcLoggerPkg "magalu.cloud/core/logger"
)

var logger = mgcLoggerPkg.NewLazy[ConstraintKind]()
