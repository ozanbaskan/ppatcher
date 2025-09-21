package main

import (
	"embed"
	"encoding/json"
	"os"
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

	if envBackend := os.Getenv("BACKEND"); envBackend != "" {
		BuildConfig.Backend = envBackend
	}
	if envExecutable := os.Getenv("EXECUTABLE"); envExecutable != "" {
		BuildConfig.Executable = envExecutable
	}
	if envColorPalette := os.Getenv("COLOR_PALETTE"); envColorPalette != "" {
		BuildConfig.ColorPalette = envColorPalette
	}
	if envMode := os.Getenv("MODE"); envMode != "" {
		BuildConfig.Mode = envMode
	}
	if envVersion := os.Getenv("VERSION"); envVersion != "" {
		BuildConfig.Version = envVersion
	}
	if envDescription := os.Getenv("DESCRIPTION"); envDescription != "" {
		BuildConfig.Description = envDescription
	}
	if envTitle := os.Getenv("TITLE"); envTitle != "" {
		BuildConfig.Title = envTitle
	}
	if envDisplayName := os.Getenv("DISPLAY_NAME"); envDisplayName != "" {
		BuildConfig.DisplayName = envDisplayName
	}
}

type Config struct {
	Backend      string `json:"backend" default:"http://localhost:3000"`
	Executable   string `json:"executable"`
	ColorPalette string `json:"colorPalette" default:"neutral"`
	Mode         string `json:"mode" default:"production"`
	Version      string `json:"version" default:"v1.0.0"`
	Description  string `json:"description" default:"Keep your files up to date"`
	Title        string `json:"title" default:"ppatcher"`
	DisplayName  string `json:"displayName" default:"PPatcher"`
	Logo         string `json:"logo"`
	Icon         string `json:"icon"`
}

func MarshalConfig(data []byte) *Config {
	var config Config
	err := json.Unmarshal(data, &config)
	if err != nil {
		panic(err)
	}
	return &config
}
