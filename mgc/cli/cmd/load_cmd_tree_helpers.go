package cmd

import (
	"fmt"
	"os"

	pflag "github.com/spf13/pflag"
	mgcSdk "magalu.cloud/sdk"
)

func getPropType(prop *mgcSdk.Schema) string {
	result := prop.Type

	if prop.Type == "array" && prop.Items != nil {
		result += fmt.Sprintf("(%v)", getPropType((*mgcSdk.Schema)(prop.Items.Value)))
	} else if len(prop.Enum) != 0 {
		result = "enum"
	}

	return result
}

func getFlagValue(flags *pflag.FlagSet, name string) (mgcSdk.Value, *pflag.Flag, error) {
	flag := flags.Lookup(name)
	if flag == nil {
		return nil, nil, os.ErrNotExist
	}

	if f, ok := flag.Value.(*anyFlagValue); ok {
		return f.Value(), flag, nil
	} else if val, err := flags.GetBool(name); err == nil {
		return val, flag, nil
	} else {
		return nil, flag, fmt.Errorf("Could not get flag value %q: %w", name, err)
	}
}
