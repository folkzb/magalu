package objects

import mgcLoggerPkg "magalu.cloud/core/logger"

var logger = mgcLoggerPkg.NewLazy[ListObjectsResponse]()
