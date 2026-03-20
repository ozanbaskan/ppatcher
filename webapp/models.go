package main

import (
	"time"
)

type User struct {
	ID        int64
	GoogleID  string
	Email     string
	Name      string
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Application struct {
	ID           int64
	UserID       int64
	Name         string
	Description  string
	ServerMode   string // ssh
	ServerHost   string
	ServerUser   string
	ServerPort   string // port the file server listens on
	SSHPort      string // SSH connection port (default 22)
	SSHKeyPath   string
	SSHPassword  string
	SSHRemoteDir string
	FilesDir     string
	BackendURL   string
	ColorPalette string
	Version      string
	Title        string
	DisplayName  string
	Executable   string
	OutputName   string
	FallbackURLs      string
	ClientDescription string
	AdminKey          string
	LogoPath          string
	IconPath          string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
