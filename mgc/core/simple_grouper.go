package core

type SimpleGrouper[T Descriptor] struct {
	SimpleDescriptor
	*GrouperLazyChildren[T]
}

func NewSimpleGrouper[T Descriptor](spec DescriptorSpec, createChildren func() ([]T, error)) *SimpleGrouper[T] {
	return &SimpleGrouper[T]{
		SimpleDescriptor{spec},
		NewGrouperLazyChildren[T](createChildren),
	}
}

// implemented by embedded GrouperLazyChildren & SimpleDescriptor
var _ Grouper = (*SimpleGrouper[Descriptor])(nil)
