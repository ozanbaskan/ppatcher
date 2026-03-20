package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

//go:embed templates/*.html
var templateFS embed.FS

//go:embed static
var staticFS embed.FS

var (
	templates *template.Template
	store     *sessions.CookieStore
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	sessionSecret := os.Getenv("SESSION_SECRET")
	if sessionSecret == "" {
		sessionSecret = "dev-secret-change-me"
	}
	store = sessions.NewCookieStore([]byte(sessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	if err := InitDB(); err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer CloseDB()

	InitOAuth()

	var err error
	templates, err = template.New("").Funcs(template.FuncMap{
		"truncate": func(s string, n int) string {
			if len(s) <= n {
				return s
			}
			return s[:n] + "..."
		},
		"maskKey": func(s string) string {
			if len(s) <= 4 {
				return "••••"
			}
			return s[:4] + "••••"
		},
		"dict": func(pairs ...interface{}) map[string]interface{} {
			m := make(map[string]interface{}, len(pairs)/2)
			for i := 0; i < len(pairs)-1; i += 2 {
				m[pairs[i].(string)] = pairs[i+1]
			}
			return m
		},
		"hasPrefix": strings.HasPrefix,
		"gaID": func() string { return os.Getenv("GA_MEASUREMENT_ID") },
	}).ParseFS(templateFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Template parse failed: %v", err)
	}

	r := mux.NewRouter()

	// Static files
	r.PathPrefix("/static/").Handler(http.FileServer(http.FS(staticFS)))

	// Public pages
	r.HandleFunc("/", handleIndex).Methods("GET")

	// Auth
	r.HandleFunc("/auth/google", handleGoogleLogin).Methods("GET")
	r.HandleFunc("/auth/google/callback", handleGoogleCallback).Methods("GET")
	r.HandleFunc("/auth/logout", handleLogout).Methods("POST")

	// Protected pages
	r.HandleFunc("/dashboard", requireAuth(handleDashboard)).Methods("GET")
	r.HandleFunc("/apps/new", requireAuth(handleNewApp)).Methods("GET")
	r.HandleFunc("/apps/new", requireAuth(handleCreateApp)).Methods("POST")
	r.HandleFunc("/apps/{id}", requireAuth(handleViewApp)).Methods("GET")
	r.HandleFunc("/apps/{id}/edit", requireAuth(handleEditApp)).Methods("GET")
	r.HandleFunc("/apps/{id}/edit", requireAuth(handleUpdateApp)).Methods("POST")
	r.HandleFunc("/apps/{id}/delete", requireAuth(handleDeleteApp)).Methods("POST")
	r.HandleFunc("/apps/{id}/version", requireAuth(handleUpdateVersion)).Methods("POST")
	r.HandleFunc("/apps/{id}/build", requireAuth(handleBuildPage)).Methods("GET")
	r.HandleFunc("/apps/{id}/build", requireAuth(handleStartBuild)).Methods("POST")
	r.HandleFunc("/apps/{id}/build/status", requireAuth(handleBuildStatus)).Methods("GET")
	r.HandleFunc("/apps/{id}/build/download/{filename}", requireAuth(handleDownloadBuild)).Methods("GET")
	r.HandleFunc("/api/build-status", requireAuth(handleGlobalBuildStatus)).Methods("GET")

	// Server deploy & test
	r.HandleFunc("/apps/{id}/test", requireAuth(handleTestConnection)).Methods("POST")
	r.HandleFunc("/apps/{id}/test-ssh", requireAuth(handleTestSSH)).Methods("POST")
	r.HandleFunc("/apps/{id}/deploy", requireAuth(handleDeploySSH)).Methods("POST")
	r.HandleFunc("/apps/{id}/verify-backend", requireAuth(handleVerifyBackend)).Methods("POST")

	r.HandleFunc("/apps/{id}/ssh-key", requireAuth(handleUploadSSHKey)).Methods("POST")
	r.HandleFunc("/apps/{id}/admin-key", requireAuth(handleGetAdminKey)).Methods("GET")
	r.HandleFunc("/apps/{id}/assets/{kind}", requireAuth(handleUploadBrandAsset)).Methods("POST")
	r.HandleFunc("/apps/{id}/assets/{kind}", requireAuth(handleGetBrandAsset)).Methods("GET")
	r.HandleFunc("/apps/{id}/fs/{path:.*}", requireAuth(handleFSProxy)).Methods("GET", "POST", "PUT", "DELETE", "OPTIONS")
	r.HandleFunc("/server-bins/rebuild", requireAuth(handleRebuildServerBins)).Methods("POST")

	// Pre-build server binaries in background on startup
	go func() {
		if err := ensureServerBins(); err != nil {
			log.Printf("Warning: failed to pre-build server binaries: %v", err)
		} else {
			log.Println("Server binaries ready")
		}
	}()

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  5 * time.Minute,
		WriteTimeout: 5 * time.Minute,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("PPatcher Web App listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
	log.Println("Server stopped")
}
