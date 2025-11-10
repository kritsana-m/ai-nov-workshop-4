package config

import (
	"fmt"

	"github.com/spf13/viper"
)

// LoadConfig reads configuration from `config.yaml` (if present) and returns the
// server port and database path. Defaults are used when the file is not found.
func LoadConfig() (int, string) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.SetDefault("server.port", 3000)
	viper.SetDefault("database.path", "data.db")
	if err := viper.ReadInConfig(); err != nil {
		// not an error — fall back to defaults
		fmt.Println("no config file found, using defaults")
	}
	return viper.GetInt("server.port"), viper.GetString("database.path")
}
