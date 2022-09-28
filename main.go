package main

import (
	"fmt"

	"github.com/PKeidel/fakeapi/admin"
	"github.com/spf13/viper"
)

func init() {
	viper.SetConfigName("config")                // name of config file (without extension)
	viper.SetConfigType("yaml")                  // REQUIRED if the config file does not have the extension in the name
	viper.AddConfigPath("/etc/fakeapi/")         // path to look for the config file in
	viper.AddConfigPath("$HOME/.config/fakeapi") // call multiple times to add many search paths
	viper.AddConfigPath(".")                     // optionally look for config in the working directory
	err := viper.ReadInConfig()                  // Find and read the config file
	if err != nil {                              // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	viper.SetDefault("admin.port", 8080)

	viper.WatchConfig()
}

func main() {
	srv := admin.AdminServer{}
	srv.StartFakeApi()
}
