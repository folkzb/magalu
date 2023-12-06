package schema

import (
	"magalu.cloud/core/utils"
)

type ObjectPropertySchema struct {
	PropName   string
	PropSchema *Schema

	Container *Schema
	Field     string // "AnyOf", "OneOf"... (Golang names). Empty ("") for root
	Index     int    // whenever processing AnyOf, OneOf... the array index. Root is always 0
}

func CollectObjectPropertySchemas(s *Schema, propName string) (schemas []ObjectPropertySchema) {
	_, _ = ForEachObjectProperty(s, func(ps ObjectPropertySchema) (run bool, err error) {
		if ps.PropName == propName {
			schemas = append(schemas, ps)
		}
		return true, nil
	})

	return schemas
}

func CollectAllObjectPropertySchemas(s *Schema) (m map[string][]ObjectPropertySchema) {
	_, _ = ForEachObjectProperty(s, func(ps ObjectPropertySchema) (run bool, err error) {
		if m == nil {
			m = map[string][]ObjectPropertySchema{}
		}
		m[ps.PropName] = append(m[ps.PropName], ps)
		return true, nil
	})

	return m
}

type ForEachObjectPropertyCb func(ps ObjectPropertySchema) (
	run bool, // if true, keep running. False early aborts the walk
	err error, // if any error should be reported
)

// JSON Schema allows properties to live in both the root level or at AllOf, AnyOf, OneOf. This visits them all.
//
// WARNING: In the case of `AnyOf` and `OneOf`, the properties may have the same name but different types/constraints!
//
// Properties are sorted before walking *AT THE SAME LEVEL*, different stack levels are not considered.
//
// Finished is true if all entries were processed. It's false if it was early aborted
func ForEachObjectProperty(s *Schema, cb ForEachObjectPropertyCb) (finished bool, err error) {
	return forEachObjectProperty(ObjectPropertySchema{Container: s}, cb)
}

func forEachObjectProperty(ps ObjectPropertySchema, cb ForEachObjectPropertyCb) (finished bool, err error) {
	finished = true
	s := ps.Container
	if s == nil {
		return
	}

	if len(s.Properties) > 0 {
		for _, pair := range utils.SortedMapIterator(s.Properties) {
			propName := pair.Key
			propSchemaRef := pair.Value
			if propSchemaRef == nil || propSchemaRef.Value == nil {
				continue
			}
			ps.PropName = propName
			ps.PropSchema = (*Schema)(propSchemaRef.Value)
			// Container is already set, Field/Index are untouched
			finished, err = cb(ps)
			if err != nil || !finished {
				return
			}
		}
	}

	finished, err = forEachObjectPropertyXOf("AnyOf", s.AnyOf, cb)
	if err != nil || !finished {
		return
	}

	finished, err = forEachObjectPropertyXOf("OneOf", s.OneOf, cb)
	if err != nil || !finished {
		return
	}

	return
}

func forEachObjectPropertyXOf(field string, refs SchemaRefs, cb ForEachObjectPropertyCb) (finished bool, err error) {
	finished = true
	if len(refs) == 0 {
		return
	}

	os := ObjectPropertySchema{Field: field}
	for i, ref := range refs {
		if ref == nil || ref.Value == nil {
			continue
		}
		os.Container = (*Schema)(ref.Value)
		os.Index = i
		finished, err = forEachObjectProperty(os, cb)
		if err != nil || !finished {
			return
		}
	}

	return
}
