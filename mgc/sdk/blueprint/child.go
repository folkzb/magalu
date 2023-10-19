package blueprint

import (
	"go.uber.org/zap"
	"magalu.cloud/core"
)

func newChild(spec *childSpec, logger *zap.SugaredLogger, refResolver *core.BoundRefPathResolver) (core.Descriptor, error) {
	if !spec.grouperSpec.isEmpty() {
		return newGrouper(spec, logger, refResolver)
	}
	return newExecutor(spec, logger, refResolver)
}
