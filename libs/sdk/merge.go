package mgc_sdk

import "fmt"

type MergeGroup struct {
	name        string
	version     string
	description string
	merge       []Grouper
}

func NewMergeGroup(name string, version string, description string, merge []Grouper) *MergeGroup {
	return &MergeGroup{name, version, description, merge}
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

func (o *MergeGroup) mergeAfter(target Grouper, start int) []Grouper {
	result := make([]Grouper, 1, len(o.merge)-start+1)
	result[0] = target

	name := target.Name()

	for i := start; i < len(o.merge); i += 1 {
		child, err := o.merge[i].GetChildByName(name)
		if err == nil {
			if group, ok := child.(Grouper); ok {
				result = append(result, group)
			}
		}
	}
	return result
}

func (o *MergeGroup) VisitChildren(visitor DescriptorVisitor) (finished bool, err error) {
	used := map[string]bool{}

	for i, c := range o.merge {
		finished, err := c.VisitChildren(func(child Descriptor) (run bool, err error) {
			name := child.Name()

			if used[name] {
				return true, nil
			}

			if group, ok := child.(Grouper); ok {
				merge := o.mergeAfter(group, i+1)
				if len(merge) > 1 {
					used[name] = true
					return visitor(&MergeGroup{
						name:        name,
						version:     group.Version(),
						description: group.Description(),
						merge:       merge,
					})
				}
			}

			return visitor(child)
		})

		if err != nil {
			return false, err
		}
		if !finished {
			return false, nil
		}
	}

	return true, nil
}

func (o *MergeGroup) GetChildByName(name string) (child Descriptor, err error) {
	var found Descriptor
	finished, err := o.VisitChildren(func(child Descriptor) (run bool, err error) {
		if child.Name() == name {
			found = child
			return false, nil
		}
		return true, nil
	})

	if err != nil {
		return nil, err
	}

	if finished {
		return nil, fmt.Errorf("Child not found: %s", name)
	}

	return found, err
}

var _ Grouper = (*MergeGroup)(nil)

// END: Grouper interface
