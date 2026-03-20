package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var db *pgxpool.Pool

func InitDB() error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://ppatcher:ppatcher@localhost:5432/ppatcher?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var err error
	db, err = pgxpool.New(ctx, dsn)
	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}

	if err := db.Ping(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if err := migrate(ctx); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Println("Database connected and migrated")
	return nil
}

func CloseDB() {
	if db != nil {
		db.Close()
	}
}

func migrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id            BIGSERIAL PRIMARY KEY,
			google_id     TEXT UNIQUE NOT NULL,
			email         TEXT UNIQUE NOT NULL,
			name          TEXT NOT NULL DEFAULT '',
			avatar_url    TEXT NOT NULL DEFAULT '',
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS applications (
			id             BIGSERIAL PRIMARY KEY,
			user_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			name           TEXT NOT NULL,
			description    TEXT NOT NULL DEFAULT '',
			server_mode    TEXT NOT NULL DEFAULT 'local',
			server_host    TEXT NOT NULL DEFAULT '',
			server_user    TEXT NOT NULL DEFAULT '',
			server_port    TEXT NOT NULL DEFAULT '3000',
			ssh_port       TEXT NOT NULL DEFAULT '22',
			ssh_key_path   TEXT NOT NULL DEFAULT '',
			ssh_password   TEXT NOT NULL DEFAULT '',
			ssh_remote_dir TEXT NOT NULL DEFAULT '~/ppatcher-server',
			files_dir      TEXT NOT NULL DEFAULT '',
			backend_url    TEXT NOT NULL DEFAULT '',
			color_palette  TEXT NOT NULL DEFAULT 'blue',
			version        TEXT NOT NULL DEFAULT '1.0.0',
			title          TEXT NOT NULL DEFAULT '',
			display_name   TEXT NOT NULL DEFAULT '',
			executable     TEXT NOT NULL DEFAULT '',
			output_name    TEXT NOT NULL DEFAULT '',
			fallback_urls  TEXT NOT NULL DEFAULT '',
			client_description TEXT NOT NULL DEFAULT '',
			admin_key      TEXT NOT NULL DEFAULT '',
			logo_path      TEXT NOT NULL DEFAULT '',
			icon_path      TEXT NOT NULL DEFAULT '',
			created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		// Migration: add new columns / rename for existing tables
		`DO $$ BEGIN
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS server_mode TEXT NOT NULL DEFAULT 'local';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS ssh_port TEXT NOT NULL DEFAULT '22';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS ssh_key_path TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS ssh_password TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS ssh_remote_dir TEXT NOT NULL DEFAULT '~/ppatcher-server';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS files_dir TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS version TEXT NOT NULL DEFAULT '1.0.0';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS title TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS display_name TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS executable TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS output_name TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS fallback_urls TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS client_description TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS admin_key TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS logo_path TEXT NOT NULL DEFAULT '';
			ALTER TABLE applications ADD COLUMN IF NOT EXISTS icon_path TEXT NOT NULL DEFAULT '';
		END $$`,
		`CREATE INDEX IF NOT EXISTS idx_applications_user_id ON applications(user_id)`,
	}

	for _, q := range queries {
		if _, err := db.Exec(ctx, q); err != nil {
			return fmt.Errorf("migration query failed: %w\nQuery: %s", err, q)
		}
	}
	return nil
}
