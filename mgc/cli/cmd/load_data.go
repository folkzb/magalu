package cmd

import (
	"fmt"

	flag "github.com/spf13/pflag"
	"magalu.cloud/core"
	mgcSdk "magalu.cloud/sdk"
)

func loadDataFromArgs(argNames, argValues []string, flags *flag.FlagSet) error {
	for i, value := range argValues {
		if err := flags.Set(argNames[i], value); err != nil {
			return err
		}
	}

	return nil
}

func loadDataFromFlags(flags *flag.FlagSet, schema *mgcSdk.Schema, dst map[string]core.Value) error {
	if flags == nil || schema == nil || dst == nil {
		return fmt.Errorf("invalid command or parameter schema")
	}

	for name, propRef := range schema.Properties {
		propSchema := propRef.Value
		val, flag, err := getFlagValue(flags, name)
		if err != nil {
			return err
		}

		if propSchema.Default != nil {
			dst[name] = propSchema.Default
		}

		if flag.Changed {
			dst[name] = val
		}
	}

	return nil
}

func loadDataFromConfig(config *mgcSdk.Config, flags *flag.FlagSet, schema *mgcSdk.Schema, dst map[string]mgcSdk.Value) error {
	for name, propRef := range schema.Properties {
		propSchema := propRef.Value
		val, flag, err := getFlagValue(flags, name)
		if err != nil {
			return err
		}

		if flag == nil {
			if propSchema.Default != nil {
				dst[name] = propSchema.Default
			}
			continue
		}

		var cfgVal any
		errCfg := config.Get(name, &cfgVal)
		if errCfg != nil {
			return errCfg
		}

		if flag.Changed || cfgVal == nil {
			if err != nil {
				return err
			}
			dst[name] = val
		} else {
			dst[name] = cfgVal
		}
	}

	return nil
}
