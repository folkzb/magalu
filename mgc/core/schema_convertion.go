package core

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/invopop/jsonschema"
)

func toCoreSchema(s *jsonschema.Schema) (*Schema, error) {
	if s == nil {
		return nil, fmt.Errorf("invalid jsonschema.Schema passed to 'toCoreSchema' function")
	}

	// The first definition in a jsonschema.Schema is the definition of the struct that was
	// turned into a schema, so we handle that. In the future, if we want to handle sub-structs
	// (like a field in a struct whose type is another struct), we'll need to deal with the
	// subsequent definitions. Because of this, only native types are allowed for now.
	var def *jsonschema.Schema
	for _, d := range s.Definitions {
		def = d
		break
	}

	if def == nil {
		// Probably means that an empty struct was passed as Params/Configs
		return &Schema{}, nil
	}

	json, err := def.MarshalJSON()
	if err != nil {
		return nil, err
	}

	newS := &openapi3.Schema{}
	err = newS.UnmarshalJSON(json)
	if err != nil {
		return nil, err
	}

	return (*Schema)(newS), nil
}
