package main

import (
	"embed"
	"encoding/json"
)

//go:embed config.json
var buildConfig embed.FS

var BuildConfig *Config

func InitConfig() {
	data, err := buildConfig.ReadFile("config.json")
	if err != nil {
		panic(err)
	}
	BuildConfig = MarshalConfig(data)
}

type Config struct {
	Backend      string `json:"backend"`
	Executable   string `json:"executable"`
	ColorPalette string `json:"colorPalette"`
}

func MarshalConfig(data []byte) *Config {
	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	return &config
}
