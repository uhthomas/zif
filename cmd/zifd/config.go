package main

import (
	"fmt"

	"github.com/fsnotify/fsnotify"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

func SetupConfig() {
	viper.SetConfigName("zifd")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.zif")
	viper.AddConfigPath("/etc/zif")

	err := viper.ReadInConfig()

	if err != nil {
		panic(fmt.Errorf("Fatal error loading config file: %s \n", err))
	}

	viper.SetDefault("bind", map[string]string{
		"zif":  "0.0.0.0:5050",
		"http": "127.0.0.1:8080",
	})

	// someday support postgresql, etc. Hence the map :)
	viper.SetDefault("database", map[string]string{
		"path": "./data/posts.db",
	})

	viper.SetDefault("tor", map[string]interface{}{
		"enabled":    true,
		"control":    10051,
		"socks":      10050,
		"cookiePath": "./tor/",
	})

	viper.SetDefault("socks", map[string]interface{}{"enabled": true, "port": 10050})

	viper.SetDefault("net", map[string]interface{}{
		"maxPeers": 100,
	})

	viper.WatchConfig()

	viper.OnConfigChange(func(e fsnotify.Event) {
		log.Info("Config file changed, reloading: ", e.Name)
	})
}
