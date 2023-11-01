package cmd

import (
	"fmt"

	flag "github.com/spf13/pflag"
	"magalu.cloud/core"
	mgcSchemaPkg "magalu.cloud/core/schema"
	mgcSdk "magalu.cloud/sdk"
)

func loadDataFromFlags(flags *flag.FlagSet, schema *mgcSdk.Schema, dst map[string]core.Value) error {
	if flags == nil || schema == nil || dst == nil {
		return fmt.Errorf("invalid command or parameter schema")
	}

	for name, propRef := range schema.Properties {
		propSchema := propRef.Value
		val, flag, err := getFlagValue(flags, name)
		if flag == nil {
			continue
		}
		if err != nil {
			return err
		}
		if val == nil && !mgcSchemaPkg.IsSchemaNullable((*core.Schema)(propSchema)) {
			continue
		}
		dst[name] = val
	}

	return nil
}

func loadDataFromConfig(config *mgcSdk.Config, flags *flag.FlagSet, schema *mgcSdk.Schema, dst map[string]mgcSdk.Value) error {
	for name := range schema.Properties {
		val, flag, err := getFlagValue(flags, name)
		if flag == nil {
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
