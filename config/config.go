package config

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	MongoDB struct {
		Host string `yaml:"host"`
		Port string `yaml:"port"`
		Name string `yaml:"name"`
	} `yaml:"mongodb"`
	Telegram struct {
		APIKey string `yaml:"api_key"`
		RoomID int64  `yaml:"room_id"`
	} `yaml:"telegram"`
}

func LoadConfig() Config {

	slog.Info("Load Config")
	f, err := os.Open("config.yaml")
	if err != nil {
		slog.Error("Can't open config.yaml", "err", err)
		os.Exit(1)
	}
	defer f.Close()

	var cfg Config
	decoder := yaml.NewDecoder(f)
	err = decoder.Decode(&cfg)
	if err != nil {
		slog.Error("load-config error to decode", "err", err)
		os.Exit(1)
	}

	return cfg

}
