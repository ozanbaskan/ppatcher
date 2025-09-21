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
	Backend      string `json:"backend"`
	Executable   string `json:"executable"`
	ColorPalette string `json:"colorPalette"`
	Mode         string `json:"mode"`
	Version      string `json:"version"`
	Description  string `json:"description"`
	Title        string `json:"title"`
	DisplayName  string `json:"displayName"`
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
