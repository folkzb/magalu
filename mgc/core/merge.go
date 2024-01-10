package core

type MergeGroup struct {
	SimpleDescriptor
	createToMerge func() []Grouper
	*GrouperLazyChildren[Descriptor]
}

func NewMergeGroup(desc DescriptorSpec, createToMerge func() []Grouper) (o *MergeGroup) {
	o = &MergeGroup{SimpleDescriptor{desc}, createToMerge, NewGrouperLazyChildren[Descriptor](func() (merged []Descriptor, err error) {
		merged, err = createChildren(o.createToMerge)
		o.createToMerge = nil
		return merged, err
	})}
	return o
}

func mergeAfter(toMerge []Grouper, target Grouper, start int) (result []Grouper) {
	result = make([]Grouper, 1, len(toMerge)-start+1)
	result[0] = target

	name := target.Name()

	for i := start; i < len(toMerge); i += 1 {
		child, err := toMerge[i].GetChildByName(name)
		if err == nil {
			if group, ok := child.(Grouper); ok {
				result = append(result, group)
			}
		}
	}
	return result
}

func createChildren(createToMerge func() []Grouper) (children []Descriptor, err error) {
	toMerge := createToMerge()
	children = make([]Descriptor, 0)
	used := make(map[string]bool)

	for i, c := range toMerge {
		finished, err := c.VisitChildren(func(child Descriptor) (run bool, err error) {
			name := child.Name()

			if used[name] {
				return true, nil
			}

			if group, ok := child.(Grouper); ok {
				merged := mergeAfter(toMerge, group, i+1)
				if len(merged) > 1 {
					child = NewMergeGroup(group.DescriptorSpec(), func() []Grouper { return merged })
				}
			}

			children = append(children, child)
			used[name] = true
			return true, nil
		})

		if err != nil {
			return nil, err
		}
		if !finished {
			return nil, err
		}
	}

	return children, nil
}

// implemented by embedded GrouperLazyChildren & SimpleDescriptor
var _ Grouper = (*MergeGroup)(nil)
