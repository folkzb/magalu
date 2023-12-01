package cmd

import (
	"fmt"
	"os"

	pflag "github.com/spf13/pflag"
	"magalu.cloud/cli/cmd/schema_flags"
	mgcSdk "magalu.cloud/sdk"
)

func getFlagValue(flags *pflag.FlagSet, name string) (mgcSdk.Value, *pflag.Flag, error) {
	flag := flags.Lookup(name)
	if flag == nil {
		return nil, nil, os.ErrNotExist
	}

	if f, ok := flag.Value.(schema_flags.SchemaFlagValue); ok {
		value, err := f.Parse()
		return value, flag, err
	}

	return nil, flag, fmt.Errorf("flag is not a schema flag %q, but %#v", name, flag)
}
