package blueprint

import (
	"fmt"

	"github.com/invopop/yaml"
	"magalu.cloud/core"
	schemaPkg "magalu.cloud/core/schema"
)

const DocumentVersion = "1.0.0"

type componentsSpec struct {
	Schemas           map[string]*schemaPkg.SchemaRef `json:"schemas"`
	ParametersSchemas map[string]*schemaPkg.SchemaRef `json:"parametersSchemas"`
	ConfigsSchemas    map[string]*schemaPkg.SchemaRef `json:"configsSchemas"`
	ResultSchemas     map[string]*schemaPkg.SchemaRef `json:"resultSchemas"`
}

type document struct {
	Blueprint  string         `json:"blueprint"`
	Url        string         `json:"url"`
	Components componentsSpec `json:"components,omitempty"`
	core.DescriptorSpec
	grouperSpec
}

func (d *document) validate() (err error) {
	if d.Blueprint != DocumentVersion {
		return fmt.Errorf("expected blueprint version %q, got %q", DocumentVersion, d.Blueprint)
	}

	err = d.DescriptorSpec.Validate()
	if err != nil {
		return &core.ChainedError{
			Name: d.DescriptorSpec.Name,
			Err:  fmt.Errorf("invalid document: %w", err),
		}
	}

	err = d.grouperSpec.validate()
	if err != nil {
		return &core.ChainedError{
			Name: d.DescriptorSpec.Name,
			Err:  fmt.Errorf("invalid document: %w", err),
		}
	}

	return nil
}

func newDocumentFromData(data []byte) (doc *document, err error) {
	doc = &document{}
	err = yaml.Unmarshal(data, doc) // use invopop/yaml instead of plain so we get UnmarshalJSON() to be used
	if err != nil {
		return nil, err
	}

	return doc, nil
}
