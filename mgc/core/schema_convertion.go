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

	json, err := s.MarshalJSON()
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
