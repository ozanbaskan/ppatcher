package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/mux"
)

// Per-app build state tracked in memory.
type buildState struct {
	mu       sync.Mutex
	log      strings.Builder
	done     bool
	errMsg   string
	files    []buildFile
	building bool
	appName  string // display name for the banner
}

type buildFile struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
}

var (
	builds   = make(map[int64]*buildState) // key: app ID
	buildsMu sync.Mutex
)

func getBuildState(appID int64) *buildState {
	buildsMu.Lock()
	defer buildsMu.Unlock()
	bs, ok := builds[appID]
	if !ok {
		bs = &buildState{}
		builds[appID] = bs
	}
	return bs
}

// ppatcherRoot returns the root directory of the ppatcher project.
// Set via PPATCHER_ROOT env var, or defaults to parent of the working directory.
func ppatcherRoot() string {
	if root := os.Getenv("PPATCHER_ROOT"); root != "" {
		return root
	}
	// Default: assume webapp runs from <ppatcher>/webapp or <ppatcher>
	cwd, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(cwd, "build-client.sh")); err == nil {
		return cwd
	}
	parent := filepath.Dir(cwd)
	if _, err := os.Stat(filepath.Join(parent, "build-client.sh")); err == nil {
		return parent
	}
	return cwd
}

// appDistDir returns the per-app directory for storing build artifacts.
func appDistDir(appID int64) string {
	return filepath.Join(ppatcherRoot(), "webapp-builds", fmt.Sprintf("app-%d", appID))
}

// generateBuildConfig writes a config.json for build-client.sh from the Application model.
func generateBuildConfig(app *Application) (string, error) {
	distDir := appDistDir(app.ID)
	if err := os.MkdirAll(distDir, 0755); err != nil {
		return "", fmt.Errorf("create dist dir: %w", err)
	}

	backendURL := app.BackendURL

	outputName := app.OutputName
	if outputName == "" {
		outputName = strings.ReplaceAll(strings.ToLower(app.Name), " ", "-") + "-patcher"
	}

	displayName := app.DisplayName
	if displayName == "" {
		displayName = app.Name
	}

	title := app.Title
	if title == "" {
		title = app.Name + " Patcher"
	}

	// Parse fallback URLs from comma-separated string
	var fallbackURLs []string
	for _, u := range strings.Split(app.FallbackURLs, ",") {
		u = strings.TrimSpace(u)
		if u != "" {
			fallbackURLs = append(fallbackURLs, u)
		}
	}

	// Use uploaded branding images if they exist on disk
	logoPath := app.LogoPath
	if logoPath != "" {
		if _, err := os.Stat(logoPath); err != nil {
			logoPath = ""
		}
	}
	iconPath := app.IconPath
	if iconPath != "" {
		if _, err := os.Stat(iconPath); err != nil {
			iconPath = ""
		}
	}

	config := map[string]interface{}{
		"backend":      backendURL,
		"fallbackUrls": fallbackURLs,
		"executable":   app.Executable,
		"colorPalette": app.ColorPalette,
		"mode":         "production",
		"version":      app.Version,
		"description":  app.ClientDescription,
		"title":        title,
		"displayName":  displayName,
		"outputName":   outputName,
		"logo":         logoPath,
		"icon":         iconPath,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}

	configPath := filepath.Join(distDir, "config.json")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return "", err
	}

	return configPath, nil
}

// ---------- Build Page ----------

type buildPageData struct {
	User     *User
	App      *Application
	Building bool
	Done     bool
	Error    string
	Log      string
	Files    []buildFile
}

func handleBuildPage(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	bs := getBuildState(app.ID)
	bs.mu.Lock()
	data := buildPageData{
		User:     user,
		App:      app,
		Building: bs.building,
		Done:     bs.done,
		Error:    bs.errMsg,
		Log:      bs.log.String(),
		Files:    bs.files,
	}
	bs.mu.Unlock()

	renderTemplate(w, "build.html", data)
}

// ---------- Start Build ----------

func handleStartBuild(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	platforms := r.FormValue("platforms")
	if platforms == "" {
		platforms = "linux/amd64"
	}

	bs := getBuildState(app.ID)
	bs.mu.Lock()
	if bs.building {
		bs.mu.Unlock()
		http.Redirect(w, r, fmt.Sprintf("/apps/%d/build", app.ID), http.StatusSeeOther)
		return
	}
	bs.log.Reset()
	bs.done = false
	bs.errMsg = ""
	bs.files = nil
	bs.building = true
	bs.appName = app.Name
	bs.mu.Unlock()

	root := ppatcherRoot()

	// Generate config.json from app settings
	configPath, err := generateBuildConfig(app)
	if err != nil {
		bs.mu.Lock()
		bs.done = true
		bs.building = false
		bs.errMsg = "Failed to generate config: " + err.Error()
		bs.mu.Unlock()
		http.Redirect(w, r, fmt.Sprintf("/apps/%d/build", app.ID), http.StatusSeeOther)
		return
	}

	distDir := appDistDir(app.ID)

	go func() {
		defer func() {
			bs.mu.Lock()
			bs.building = false
			bs.mu.Unlock()
		}()

		// Copy the per-app config.json to the project root so it gets embedded
		// by Go's //go:embed directive. This ensures the baked-in config always
		// matches the current app, even if ldflags have quoting issues.
		rootConfig := filepath.Join(root, "config.json")
		if data, err := os.ReadFile(configPath); err == nil {
			os.WriteFile(rootConfig, data, 0644)
		}

		scriptPath := filepath.Join(root, "build-client.sh")
		cmd := exec.Command("bash", scriptPath,
			"--config", configPath,
			"--platforms", platforms,
		)
		cmd.Dir = root

		output, err := cmd.CombinedOutput()

		bs.mu.Lock()
		bs.log.Write(output)
		bs.done = true
		if err != nil {
			bs.errMsg = err.Error()
		}
		bs.mu.Unlock()

		// Determine output name
		outputName := app.OutputName
		if outputName == "" {
			outputName = strings.ReplaceAll(strings.ToLower(app.Name), " ", "-") + "-patcher"
		}

		// Copy built binaries to per-app dist dir
		os.MkdirAll(filepath.Join(distDir, "dist"), 0755)
		matches, _ := filepath.Glob(filepath.Join(root, "build", "bin", outputName+"*"))
		var builtFiles []buildFile
		for _, m := range matches {
			data, ferr := os.ReadFile(m)
			if ferr != nil {
				continue
			}
			dest := filepath.Join(distDir, "dist", filepath.Base(m))
			os.WriteFile(dest, data, 0755)
			builtFiles = append(builtFiles, buildFile{
				Name: filepath.Base(m),
				Size: int64(len(data)),
			})
		}

		bs.mu.Lock()
		bs.files = builtFiles
		bs.mu.Unlock()
	}()

	http.Redirect(w, r, fmt.Sprintf("/apps/%d/build", app.ID), http.StatusSeeOther)
}

// ---------- Build Status (JSON) ----------

func handleBuildStatus(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	bs := getBuildState(app.ID)
	bs.mu.Lock()
	defer bs.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"log":      bs.log.String(),
		"done":     bs.done,
		"building": bs.building,
		"error":    bs.errMsg,
		"files":    bs.files,
	})
}

// handleGlobalBuildStatus returns the active build for the current user (if any),
// so a persistent banner can be shown on any page.
func handleGlobalBuildStatus(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	if user == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"building": false})
		return
	}

	// Find all app IDs for this user
	rows, err := db.Query(r.Context(),
		`SELECT id FROM applications WHERE user_id = $1`, user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"building": false})
		return
	}
	defer rows.Close()

	buildsMu.Lock()
	defer buildsMu.Unlock()

	for rows.Next() {
		var appID int64
		if err := rows.Scan(&appID); err != nil {
			continue
		}
		bs, ok := builds[appID]
		if !ok {
			continue
		}
		bs.mu.Lock()
		if bs.building || bs.done {
			resp := map[string]interface{}{
				"app_id":   appID,
				"app_name": bs.appName,
				"building": bs.building,
				"done":     bs.done,
				"error":    bs.errMsg,
				"files":    bs.files,
			}
			bs.mu.Unlock()
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		bs.mu.Unlock()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"building": false})
}

// ---------- Download Built File ----------

func handleDownloadBuild(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	vars := mux.Vars(r)
	filename := vars["filename"]

	// Sanitize filename to prevent path traversal
	filename = filepath.Base(filename)
	if filename == "." || filename == ".." || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		http.NotFound(w, r)
		return
	}

	filePath := filepath.Join(appDistDir(app.ID), "dist", filename)
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", filename))
	log.Printf("Serving build download: %s for app %d", filename, app.ID)
	http.ServeFile(w, r, filePath)
}
