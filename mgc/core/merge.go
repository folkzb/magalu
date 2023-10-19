package core

import "errors"

type MergeGroup struct {
	name        string
	version     string
	description string
	toMerge     []Grouper
	*GrouperLazyChildren[Descriptor]
}

func NewMergeGroup(name string, version string, description string, toMerge []Grouper) (o *MergeGroup) {
	o = &MergeGroup{name, version, description, toMerge, NewGrouperLazyChildren[Descriptor](func() (merged []Descriptor, err error) {
		merged, err = createChildren(o.toMerge)
		o.toMerge = nil
		return merged, err
	})}
	return o
}

func (o *MergeGroup) Add(child Grouper) error {
	if o.toMerge == nil {
		return errors.New("cannot add children after the group's children were accessed")
	}
	o.toMerge = append(o.toMerge, child)
	return nil
}

// BEGIN: Descriptor interface:

func (o *MergeGroup) Name() string {
	return o.name
}

func (o *MergeGroup) Version() string {
	return o.version
}

func (o *MergeGroup) Description() string {
	return o.description
}

// END: Descriptor interface

// BEGIN: Grouper interface:

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

func createChildren(toMerge []Grouper) (children []Descriptor, err error) {
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
					child = NewMergeGroup(
						name,
						group.Version(),
						group.Description(),
						merged,
					)
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

// implemented by embedded GrouperLazyChildren
var _ Grouper = (*MergeGroup)(nil)

// END: Grouper interface
