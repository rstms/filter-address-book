package main

import (
	"github.com/spf13/viper"
)

func Configure(configFile string) error {
	viper.SetDefault("url", "http://127.0.0.1:2016/filterctl/")
	viper.SetConfigType("yaml")
	viper.SetConfigFile(configFile)
	return viper.ReadInConfig()
}
