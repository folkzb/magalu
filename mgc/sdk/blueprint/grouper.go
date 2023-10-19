package blueprint

import (
	"go.uber.org/zap"
	"magalu.cloud/core"
)

func newGrouper(spec *childSpec, logger *zap.SugaredLogger, refResolver *core.BoundRefPathResolver) (g core.Grouper, err error) {
	logger = logger.Named(spec.Name)
	return core.NewSimpleGrouper(
		spec.DescriptorSpec,
		func() ([]core.Descriptor, error) {
			return createGroupChildren(spec.Children, logger, refResolver)
		},
	), nil
}

func createGroupChildren(childrenSpecs []*childSpec, logger *zap.SugaredLogger, refResolver *core.BoundRefPathResolver) (children []core.Descriptor, err error) {
	children = make([]core.Descriptor, 0, len(childrenSpecs))

	for _, spec := range childrenSpecs {
		child, err := newChild(spec, logger, refResolver)
		if err != nil {
			logger.Errorw("failed to create child", "child", spec, "error", err)
			continue
		}
		children = append(children, child)
	}

	return children, nil
}
