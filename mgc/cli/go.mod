module magalu.cloud/cli

go 1.20

require (
	github.com/spf13/cobra v1.7.0
	github.com/spf13/pflag v1.0.5
	golang.org/x/exp v0.0.0-20230713183714-613f0c0eb8a1
	magalu.cloud/core v0.0.0-unversioned // indirect
	magalu.cloud/sdk v0.0.0-unversioned
)

require (
	github.com/getkin/kin-openapi v0.118.0 // indirect
	github.com/go-openapi/jsonpointer v0.19.5 // indirect
	github.com/go-openapi/swag v0.19.5 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/invopop/yaml v0.1.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mohae/deepcopy v0.0.0-20170929034955-c48cc78d4826 // indirect
	github.com/perimeterx/marshmallow v1.1.4 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace magalu.cloud/core => ../core

replace magalu.cloud/sdk => ../sdk