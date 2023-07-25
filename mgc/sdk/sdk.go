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

// The Context is created with the following values:
// - use GrouperFromContext() to retrieve Sdk.Group() (root group)
func (o *Sdk) NewContext() context.Context {
	var ctx = context.Background()
	ctx = core.NewGrouperContext(ctx, o.Group())
	return ctx
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
