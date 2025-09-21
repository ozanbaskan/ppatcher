package main

import (
	"embed"
	"encoding/json"
	"os"
)

//go:embed config.json
var buildConfig embed.FS

var BuildConfig *Config

// Variables that can be set via -ldflags
var (
	Built             = "false"
	DefaultBackend    = "http://localhost:3000"
	DefaultExecutable = "your-game-executable"
	DefaultPalette    = "neutral"
	DefaultMode       = "production"
	DefaultVersion    = "1.0.0"
	DefaultDesc       = "Keep your files up to date"
	DefaultTitle      = "ppatcher"
	DefaultDisplay    = "PPatcher"
)

func InitConfig() {
	if Built == "true" {
		BuildConfig = &Config{
			Backend:      DefaultBackend,
			Executable:   DefaultExecutable,
			ColorPalette: DefaultPalette,
			Mode:         DefaultMode,
			Version:      DefaultVersion,
			Description:  DefaultDesc,
			Title:        DefaultTitle,
			DisplayName:  DefaultDisplay,
		}
		return
	}

	configFile := "config.json"
	if envConfig := os.Getenv("CONFIG_FILE"); envConfig != "" {
		configFile = envConfig
	}

	data, err := buildConfig.ReadFile(configFile)
	if err != nil {
		BuildConfig = &Config{
			Backend:      DefaultBackend,
			Executable:   DefaultExecutable,
			ColorPalette: DefaultPalette,
			Mode:         DefaultMode,
			Version:      DefaultVersion,
			Description:  DefaultDesc,
			Title:        DefaultTitle,
			DisplayName:  DefaultDisplay,
		}
		return
	}
	BuildConfig = MarshalConfig(data)

	// Env overrides
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
	if err := json.Unmarshal(data, &config); err != nil {
		panic(err)
	}
	// Fill in defaults if missing
	if config.Backend == "" {
		config.Backend = DefaultBackend
	}
	if config.Executable == "" {
		config.Executable = DefaultExecutable
	}
	if config.ColorPalette == "" {
		config.ColorPalette = DefaultPalette
	}
	if config.Mode == "" {
		config.Mode = DefaultMode
	}
	if config.Version == "" {
		config.Version = DefaultVersion
	}
	if config.Description == "" {
		config.Description = DefaultDesc
	}
	if config.Title == "" {
		config.Title = DefaultTitle
	}
	if config.DisplayName == "" {
		config.DisplayName = DefaultDisplay
	}
	return &config
}
