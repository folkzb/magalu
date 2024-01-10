package core

func NewStaticGroup(spec DescriptorSpec, createChildren func() []Descriptor) Grouper {
	return NewSimpleGrouper(
		spec,
		func() ([]Descriptor, error) { return createChildren(), nil },
	)
}
