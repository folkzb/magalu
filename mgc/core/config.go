package core

import (
	"context"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type Config struct{}

const (
	CONFIG_NAME = "config"
	CONFIG_TYPE = "yaml"
	CONFIG_PATH = "$HOME/.mgc"
)

var configKey contextKey = "magalu.cloud/core/Config"

func NewConfigContext(parent context.Context, config *Config) context.Context {
	return context.WithValue(parent, configKey, config)
}

func ConfigFromContext(ctx context.Context) *Config {
	c, _ := ctx.Value(configKey).(*Config)
	return c
}

func NewConfig() *Config {
	viper.SetConfigName(CONFIG_NAME)
	viper.SetConfigType(CONFIG_TYPE)
	viper.AddConfigPath(CONFIG_PATH)
	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err != nil {
		return &Config{}
	}
	return &Config{}
}

func (c *Config) BindPFlag(key string, flag *pflag.Flag) error {
	return viper.BindPFlag(key, flag)
}

func (c *Config) IsSet(key string) bool {
	return viper.IsSet(key)
}

func (c *Config) Get(key string) any {
	return viper.Get(key)
}

func (c *Config) Set(key string, value interface{}) {
	viper.Set(key, value)
}
