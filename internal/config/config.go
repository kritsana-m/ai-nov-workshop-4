package config

import (
	"fmt"

	"github.com/spf13/viper"
)

func LoadConfig() (int, string) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetDefault("server.port", 3000)
	viper.SetDefault("database.path", "data.db")
	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("No config file found, using defaults")
	}
	return viper.GetInt("server.port"), viper.GetString("database.path")
}
