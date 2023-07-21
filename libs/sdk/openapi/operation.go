package mgc_openapi

import (
	"fmt"
	"mgc_sdk"
	"regexp"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Parameter

type Parameter struct {
	ref             *openapi3.Parameter
	extensionPrefix *string
}

// BEGIN: Parameter interface:

func (p *Parameter) Name() string {
	return getNameExtension(p.extensionPrefix, p.Schema().Extensions, p.ref.Name)
}

func (p *Parameter) Description() string {
	return p.ref.Description
}

func (p *Parameter) Required() bool {
	return p.ref.Required
}

func (p *Parameter) Schema() *mgc_sdk.Schema {
	return (*mgc_sdk.Schema)(p.ref.Schema.Value)
}

func (p *Parameter) Examples() []mgc_sdk.Example {
	exampleCapacity := len(p.ref.Examples) + 1
	examples := make([]mgc_sdk.Example, 0, exampleCapacity)

	if p.ref.Example != nil {
		examples = append(examples, p.ref.Example)
	}

	for _, example := range p.ref.Examples {
		examples = append(examples, example.Value)
	}

	return examples
}

var _ mgc_sdk.Parameter = (*Parameter)(nil)

// Source -> Module -> Resource -> Operation

// Operation

type Operation struct {
	key             string
	method          string
	path            *openapi3.PathItem
	operation       *openapi3.Operation
	doc             *openapi3.T
	parameters      *map[string]mgc_sdk.Parameter
	configs         *map[string]mgc_sdk.Config
	extensionPrefix *string
}

// BEGIN: Descriptor interface:

var openAPIPathArgRegex = regexp.MustCompile("[{](?P<name>[^}]+)[}]")

func getActionName(httpMethod string, pathName string) string {
	name := []string{string(httpMethod)}
	hasArgs := false

	for _, pathEntry := range strings.Split(pathName, "/") {
		match := openAPIPathArgRegex.FindStringSubmatch(pathEntry)
		for i, substr := range match {
			if openAPIPathArgRegex.SubexpNames()[i] == "name" {
				name = append(name, substr)
				hasArgs = true
			}
		}
		if len(match) == 0 && hasArgs {
			name = append(name, pathEntry)
		}
	}

	return strings.Join(name, "-")
}

func (o *Operation) Name() string {
	name := getNameExtension(o.extensionPrefix, o.operation.Extensions, "")
	if name == "" {
		name = getActionName(o.method, o.key)
	}
	return name
}

func (o *Operation) Version() string {
	return ""
}

func (o *Operation) Description() string {
	return o.operation.Description
}

// END: Descriptor interface

// BEGIN: Executor interface:

func addParameters(dst map[string]mgc_sdk.Parameter, src openapi3.Parameters, extensionPrefix *string) {
	// likely filter by location, like header/cookie are config
	for _, ref := range src {
		p := &Parameter{ref: ref.Value, extensionPrefix: extensionPrefix}
		dst[p.Name()] = p
	}
}

func (o *Operation) Parameters() map[string]mgc_sdk.Parameter {
	if o.parameters == nil {
		p := map[string]mgc_sdk.Parameter{}

		addParameters(p, o.path.Parameters, o.extensionPrefix)
		addParameters(p, o.operation.Parameters, o.extensionPrefix)

		o.parameters = &p
	}
	return *o.parameters
}

func (o *Operation) Configs() map[string]mgc_sdk.Config {
	if o.configs == nil {
		// TODO: convert and save
		// likely filter by location, like header/cookie are config?
		return map[string]mgc_sdk.Config{}
	}
	return *o.configs
}

func (o *Operation) Execute(parameters map[string]mgc_sdk.Value, configs map[string]mgc_sdk.Value) (result mgc_sdk.Value, err error) {
	// load definitions if not done yet
	p := o.Parameters()
	c := o.Configs()

	fmt.Printf("TODO: execute: %v %v\ninput: p=%v; c=%v\ndefinitions: p=%v; c=%v\n", o.method, o.key, parameters, configs, p, c)

	return nil, fmt.Errorf("not implemented")
}

var _ mgc_sdk.Executor = (*Operation)(nil)

// END: Executor interface
