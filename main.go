package main

import (
	"fmt"
	"log"

	"github.com/PKeidel/fakeapi/admin"
	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

var vip *viper.Viper

func init() {
	vip = viper.New()
	vip.SetConfigName("config")                // name of config file (without extension)
	vip.SetConfigType("yaml")                  // REQUIRED if the config file does not have the extension in the name
	vip.AddConfigPath("/etc/fakeapi/")         // path to look for the config file in
	vip.AddConfigPath("$HOME/.config/fakeapi") // call multiple times to add many search paths
	vip.AddConfigPath(".")                     // optionally look for config in the working directory
	err := vip.ReadInConfig()                  // Find and read the config file
	if err != nil {                            // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %w", err))
	}

	vip.SetDefault("admin.port", "127.0.0.1:8080")
	vip.SetDefault("admin.username", "admin")
	vip.SetDefault("admin.password", "admin")
	vip.SetDefault("logging.metrics.influx.enabled", false)
	vip.SetDefault("logging.metrics.influx.uri", "http://influx:8086")
	vip.SetDefault("logging.metrics.influx.token", "token")
	vip.SetDefault("logging.metrics.influx.org", "org")
	vip.SetDefault("logging.metrics.influx.bucket", "bucket")

	vip.OnConfigChange(func(in fsnotify.Event) {
		log.Println("Main config has changed and was reloaded")
	})
	vip.WatchConfig()
}

func main() {
	fmt.Println(vip.GetStringSlice("fakeapi.openapi"))

	srv := admin.NewAdminServer(vip)
	srv.StartFakeApi()

	defer srv.Close()
}
