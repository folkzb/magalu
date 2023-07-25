package sdk

import (
	"context"
	"os"
	"path/filepath"

	"magalu.cloud/core"
	"magalu.cloud/sdk/openapi"
	"magalu.cloud/sdk/static"
)

// Re-exports from Core
type Descriptor = core.Descriptor
type DescriptorVisitor = core.DescriptorVisitor
type Example = core.Example
type Executor = core.Executor
type Grouper = core.Grouper
type Schema = core.Schema
type Value = core.Value

type Sdk struct {
	group *core.MergeGroup
}

func NewSdk() *Sdk {
	return &Sdk{}
}

func (o *Sdk) NewContext() context.Context {
	// when Auth and others are added, just nest `context.WithValue()`
	// but be aware that one needs to retrieve it from static/ and openapi/ (subpackages)
	// without causing a cycle.
	// At https://pkg.go.dev/context#Context one can see how we should proceed,
	// we should add core.Auth.NewContext(parentCtx) and then core.Auth.FromContext(ctx)
	// to retrieve, using an unexported key.
	// if these functions + key is in core, there is no dep cycle
	return context.Background()
}

func (o *Sdk) newOpenApiSource() *openapi.Source {
	// TODO: are these going to be fixed? configurable?
	extensionPrefix := "x-cli"
	openApiDir := os.Getenv("MGC_SDK_OPENAPI_DIR")
	if openApiDir == "" {
		cwd, err := os.Getwd()
		if err == nil {
			openApiDir = filepath.Join(cwd, "openapis")
		}
	}

	return &openapi.Source{
		Dir:             openApiDir,
		ExtensionPrefix: &extensionPrefix,
	}
}

func (o *Sdk) Group() core.Grouper {
	if o.group == nil {
		o.group = core.NewMergeGroup(
			"MagaLu Cloud",
			"1.0",
			"All MagaLu Groups & Executors",
			[]core.Grouper{
				static.NewGroup(),
				o.newOpenApiSource(),
			},
		)
	}
	return o.group
}
