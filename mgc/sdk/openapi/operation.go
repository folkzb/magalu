package openapi

import (
	"fmt"
	"regexp"
	"strings"

	"magalu.cloud/core"

	"github.com/getkin/kin-openapi/openapi3"
	"golang.org/x/exp/slices"
)

// Source -> Module -> Resource -> Operation

// Operation

type Operation struct {
	key           string
	method        string
	path          *openapi3.PathItem
	operation     *openapi3.Operation
	doc           *openapi3.T
	paramsSchema  *core.Schema
	configsSchema *core.Schema
	// TODO: configsMapping map[string]...
	extensionPrefix *string
	servers         openapi3.Servers
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
	return getDescriptionExtension(o.extensionPrefix, o.operation.Extensions, o.operation.Description)
}

// END: Descriptor interface

// BEGIN: Executor interface:

func addParameters(schema *core.Schema, parameters openapi3.Parameters, extensionPrefix *string, locations []string) {
	for _, ref := range parameters {
		parameter := ref.Value

		if !slices.Contains(locations, parameter.In) {
			continue
		}

		paramSchemaRef := parameter.Schema
		paramSchema := paramSchemaRef.Value

		name := getNameExtension(extensionPrefix, paramSchema.Extensions, parameter.Name)

		desc := getDescriptionExtension(extensionPrefix, parameter.Extensions, parameter.Description)
		if desc == "" {
			desc = getDescriptionExtension(extensionPrefix, paramSchema.Extensions, paramSchema.Description)
		}

		if desc != "" && paramSchema.Description != desc {
			// copy, never modify parameter stuff
			newSchema := *paramSchema
			newSchema.Description = desc
			paramSchema = &newSchema

			newSchemaRef := *paramSchemaRef
			newSchemaRef.Value = paramSchema
			paramSchemaRef = &newSchemaRef
		}

		schema.Properties[name] = paramSchemaRef

		if parameter.Required && !slices.Contains(schema.Required, name) {
			schema.Required = append(schema.Required, name)
		}
	}
}

func addRequestBodyParameters(schema *core.Schema, rbr *openapi3.RequestBodyRef, extensionPrefix *string) {
	if rbr == nil {
		return
	}

	rb := rbr.Value
	mt := rb.Content.Get("application/json")
	if mt == nil {
		return
	}

	content := mt.Schema.Value
	if content == nil {
		return
	}

	for name, ref := range content.Properties {
		parameter := ref.Value
		name = getNameExtension(extensionPrefix, parameter.Extensions, name)

		for {
			_, exists := schema.Properties[name]
			if exists {
				name = "req-" + name
			} else {
				break
			}
		}

		schema.Properties[name] = ref

		if slices.Contains(content.Required, name) && !slices.Contains(schema.Required, name) {
			schema.Required = append(schema.Required, name)
		}
	}
}

var (
	parametersLocations = []string{openapi3.ParameterInPath, openapi3.ParameterInQuery}
	configLocations     = []string{openapi3.ParameterInHeader, openapi3.ParameterInCookie}
)

func (o *Operation) ParametersSchema() *core.Schema {
	if o.paramsSchema == nil {
		rootSchema := core.NewObjectSchema(map[string]*core.Schema{}, []string{})

		addParameters(rootSchema, o.path.Parameters, o.extensionPrefix, parametersLocations)
		addParameters(rootSchema, o.operation.Parameters, o.extensionPrefix, parametersLocations)
		addRequestBodyParameters(rootSchema, o.operation.RequestBody, o.extensionPrefix)

		o.paramsSchema = rootSchema
	}
	return o.paramsSchema
}

func (o *Operation) ConfigsSchema() *core.Schema {
	if o.configsSchema == nil {
		rootSchema := core.NewObjectSchema(map[string]*core.Schema{}, []string{})

		addParameters(rootSchema, o.path.Parameters, o.extensionPrefix, configLocations)
		addParameters(rootSchema, o.operation.Parameters, o.extensionPrefix, configLocations)

		o.configsSchema = rootSchema
	}
	return o.configsSchema
}

func getServerURL(servers openapi3.Servers, path string, configs map[string]core.Value) (string, error) {
	var s *openapi3.Server
	if len(servers) > 0 {
		s = servers[0]
	}

	if s == nil {
		return "", fmt.Errorf("no available servers in spec")
	}

	// TODO: Map Server Variables to URL based on received config
	return s.URL + path, nil
}

func (o *Operation) Execute(parameters map[string]core.Value, configs map[string]core.Value) (result core.Value, err error) {
	// load definitions if not done yet
	parametersSchema := o.ParametersSchema()
	configsSchema := o.ConfigsSchema()

	if err = parametersSchema.VisitJSON(parameters); err != nil {
		return nil, err
	}

	if err = configsSchema.VisitJSON(configs); err != nil {
		return nil, err
	}

	serverURL, _ := getServerURL(o.servers, o.key, configs)

	fmt.Println("server: " + serverURL)
	fmt.Printf("TODO: execute: %v %v\ninput: p=%v; c=%v\ndefinitions: p=%v; c=%v\n", o.method, o.key, parameters, configs, parametersSchema.Properties, configsSchema)

	return nil, fmt.Errorf("not implemented")
}

var _ core.Executor = (*Operation)(nil)

// END: Executor interface
