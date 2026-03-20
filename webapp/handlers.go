package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gorilla/mux"
)

// ---------- Page Data Structs ----------

type indexData struct {
	User *User
}

type dashboardData struct {
	User *User
	Apps []Application
}

type appFormData struct {
	User   *User
	App    *Application
	Errors map[string]string
}

type appViewData struct {
	User *User
	App  *Application
}

// ---------- Index ----------

func handleIndex(w http.ResponseWriter, r *http.Request) {
	user := optionalUser(r)
	if user != nil {
		http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
		return
	}
	renderTemplate(w, "index.html", indexData{User: user})
}

// ---------- Dashboard ----------

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	rows, err := db.Query(r.Context(),
		`SELECT id, user_id, name, description, server_mode, server_host, server_user, server_port,
		        ssh_port, ssh_key_path, ssh_password, ssh_remote_dir, files_dir,
		        backend_url, color_palette, version, title, display_name, executable, output_name,
		        fallback_urls, logo_path, icon_path, created_at, updated_at
		 FROM applications WHERE user_id = $1 ORDER BY updated_at DESC`, user.ID)
	if err != nil {
		log.Printf("Query apps error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var apps []Application
	for rows.Next() {
		var a Application
		if err := rows.Scan(&a.ID, &a.UserID, &a.Name, &a.Description, &a.ServerMode,
			&a.ServerHost, &a.ServerUser, &a.ServerPort,
			&a.SSHPort, &a.SSHKeyPath, &a.SSHPassword, &a.SSHRemoteDir, &a.FilesDir,
			&a.BackendURL, &a.ColorPalette, &a.Version, &a.Title, &a.DisplayName, &a.Executable, &a.OutputName,
			&a.FallbackURLs, &a.LogoPath, &a.IconPath, &a.CreatedAt, &a.UpdatedAt); err != nil {
			log.Printf("Scan app error: %v", err)
			continue
		}
		apps = append(apps, a)
	}

	renderTemplate(w, "dashboard.html", dashboardData{User: user, Apps: apps})
}

// ---------- Create App ----------

func handleNewApp(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	renderTemplate(w, "app_form.html", appFormData{
		User: user,
		App: &Application{
			ServerMode:   "ssh",
			ServerPort:   "3000",
			SSHPort:      "22",
			SSHRemoteDir: "~/ppatcher-server",
			ColorPalette: "blue",
			Version:      "1.0.0",
		},
		Errors: map[string]string{},
	})
}

func handleCreateApp(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)

	app := parseAppFromForm(r)

	isAjax := r.Header.Get("X-Requested-With") == "XMLHttpRequest"

	errors := validateApp(app, isAjax)
	if len(errors) > 0 {
		if isAjax {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "errors": errors})
			return
		}
		renderTemplate(w, "app_form.html", appFormData{User: user, App: app, Errors: errors})
		return
	}

	err := db.QueryRow(r.Context(),
		`INSERT INTO applications (user_id, name, description, server_mode, server_host, server_user, server_port,
		 ssh_port, ssh_key_path, ssh_password, ssh_remote_dir, files_dir, backend_url, color_palette,
		 version, title, display_name, executable, output_name, fallback_urls, client_description, logo_path, icon_path)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23) RETURNING id`,
		user.ID, app.Name, app.Description, app.ServerMode, app.ServerHost, app.ServerUser, app.ServerPort,
		app.SSHPort, app.SSHKeyPath, app.SSHPassword, app.SSHRemoteDir, app.FilesDir, app.BackendURL, app.ColorPalette,
		app.Version, app.Title, app.DisplayName, app.Executable, app.OutputName, app.FallbackURLs, app.ClientDescription, app.LogoPath, app.IconPath,
	).Scan(&app.ID)
	if err != nil {
		log.Printf("Insert app error: %v", err)
		if isAjax {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Internal error"})
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if isAjax {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "id": app.ID})
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ---------- View App ----------

func handleViewApp(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	renderTemplate(w, "app_view.html", appViewData{User: user, App: app})
}

// ---------- Edit App ----------

func handleEditApp(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	renderTemplate(w, "app_form.html", appFormData{User: user, App: app, Errors: map[string]string{}})
}

func handleUpdateApp(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	updated := parseAppFromForm(r)
	app.Name = updated.Name
	app.Description = updated.Description
	app.ServerMode = updated.ServerMode
	app.ServerHost = updated.ServerHost
	app.ServerUser = updated.ServerUser
	app.ServerPort = updated.ServerPort
	app.SSHPort = updated.SSHPort
	if updated.SSHKeyPath != "" {
		app.SSHKeyPath = updated.SSHKeyPath
	}
	app.SSHPassword = updated.SSHPassword
	app.SSHRemoteDir = updated.SSHRemoteDir
	app.FilesDir = updated.FilesDir
	app.BackendURL = updated.BackendURL
	app.ColorPalette = updated.ColorPalette
	app.Version = updated.Version
	app.Title = updated.Title
	app.DisplayName = updated.DisplayName
	app.Executable = updated.Executable
	app.OutputName = updated.OutputName
	app.FallbackURLs = updated.FallbackURLs
	app.ClientDescription = updated.ClientDescription
	if updated.LogoPath != "" {
		app.LogoPath = updated.LogoPath
	}
	if updated.IconPath != "" {
		app.IconPath = updated.IconPath
	}

	isAjaxUpdate := r.Header.Get("X-Requested-With") == "XMLHttpRequest"
	errors := validateApp(app, isAjaxUpdate)
	if len(errors) > 0 {
		if isAjaxUpdate {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(400)
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "errors": errors})
			return
		}
		renderTemplate(w, "app_form.html", appFormData{User: user, App: app, Errors: errors})
		return
	}

	_, err = db.Exec(r.Context(),
		`UPDATE applications SET name=$1, description=$2, server_mode=$3, server_host=$4, server_user=$5,
		 server_port=$6, ssh_port=$7, ssh_key_path=$8, ssh_password=$9, ssh_remote_dir=$10,
		 files_dir=$11, backend_url=$12, color_palette=$13,
		 version=$14, title=$15, display_name=$16, executable=$17, output_name=$18, fallback_urls=$19, client_description=$20, logo_path=$21, icon_path=$22, updated_at=$23
		 WHERE id=$24 AND user_id=$25`,
		app.Name, app.Description, app.ServerMode, app.ServerHost, app.ServerUser,
		app.ServerPort, app.SSHPort, app.SSHKeyPath, app.SSHPassword, app.SSHRemoteDir,
		app.FilesDir, app.BackendURL, app.ColorPalette,
		app.Version, app.Title, app.DisplayName, app.Executable, app.OutputName, app.FallbackURLs, app.ClientDescription, app.LogoPath, app.IconPath, time.Now(), app.ID, user.ID,
	)
	if err != nil {
		log.Printf("Update app error: %v", err)
		if isAjaxUpdate {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(500)
			json.NewEncoder(w).Encode(map[string]interface{}{"ok": false, "error": "Internal error"})
			return
		}
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	if isAjaxUpdate {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"ok": true, "id": app.ID})
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ---------- Update Version ----------

func handleUpdateVersion(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	var req struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, 400, apiResponse{Error: "Invalid request"})
		return
	}
	version := strings.TrimSpace(req.Version)
	if version == "" {
		writeJSON(w, 400, apiResponse{Error: "Version is required"})
		return
	}

	_, err = db.Exec(r.Context(),
		`UPDATE applications SET version=$1, updated_at=$2 WHERE id=$3 AND user_id=$4`,
		version, time.Now(), app.ID, user.ID,
	)
	if err != nil {
		log.Printf("Update version error: %v", err)
		writeJSON(w, 500, apiResponse{Error: "Internal error"})
		return
	}

	writeJSON(w, 200, apiResponse{OK: true, Data: version})
}

// ---------- Delete App ----------

func handleDeleteApp(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	_, err = db.Exec(r.Context(),
		`DELETE FROM applications WHERE id = $1 AND user_id = $2`, app.ID, user.ID)
	if err != nil {
		log.Printf("Delete app error: %v", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/dashboard", http.StatusSeeOther)
}

// ---------- Helpers ----------

func getAppForUser(r *http.Request, userID int64) (*Application, error) {
	vars := mux.Vars(r)
	id := vars["id"]

	var a Application
	err := db.QueryRow(r.Context(),
		`SELECT id, user_id, name, description, server_mode, server_host, server_user, server_port,
		        ssh_port, ssh_key_path, ssh_password, ssh_remote_dir, files_dir,
		        backend_url, color_palette, version, title, display_name, executable, output_name,
	        fallback_urls, client_description, admin_key, logo_path, icon_path, created_at, updated_at
	 FROM applications WHERE id = $1 AND user_id = $2`, id, userID,
	).Scan(&a.ID, &a.UserID, &a.Name, &a.Description, &a.ServerMode,
		&a.ServerHost, &a.ServerUser, &a.ServerPort,
		&a.SSHPort, &a.SSHKeyPath, &a.SSHPassword, &a.SSHRemoteDir, &a.FilesDir,
		&a.BackendURL, &a.ColorPalette, &a.Version, &a.Title, &a.DisplayName, &a.Executable, &a.OutputName,
		&a.FallbackURLs, &a.ClientDescription, &a.AdminKey, &a.LogoPath, &a.IconPath, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func parseAppFromForm(r *http.Request) *Application {
	app := &Application{
		Name:         strings.TrimSpace(r.FormValue("name")),
		Description:  strings.TrimSpace(r.FormValue("description")),
		ServerMode:   strings.TrimSpace(r.FormValue("server_mode")),
		ServerHost:   strings.TrimSpace(r.FormValue("server_host")),
		ServerUser:   strings.TrimSpace(r.FormValue("server_user")),
		ServerPort:   strings.TrimSpace(r.FormValue("server_port")),
		SSHPort:      strings.TrimSpace(r.FormValue("ssh_port")),
		SSHKeyPath:   strings.TrimSpace(r.FormValue("ssh_key_path")),
		SSHPassword:  r.FormValue("ssh_password"),
		SSHRemoteDir: strings.TrimSpace(r.FormValue("ssh_remote_dir")),
		FilesDir:     strings.TrimSpace(r.FormValue("files_dir")),
		BackendURL:   strings.TrimSpace(r.FormValue("backend_url")),
		ColorPalette: strings.TrimSpace(r.FormValue("color_palette")),
		Version:      strings.TrimSpace(r.FormValue("version")),
		Title:        strings.TrimSpace(r.FormValue("title")),
		DisplayName:  strings.TrimSpace(r.FormValue("display_name")),
		Executable:   strings.TrimSpace(r.FormValue("executable")),
		OutputName:   strings.TrimSpace(r.FormValue("output_name")),
		FallbackURLs:      strings.TrimSpace(r.FormValue("fallback_urls")),
		ClientDescription: strings.TrimSpace(r.FormValue("client_description")),
		LogoPath:          strings.TrimSpace(r.FormValue("logo_path")),
		IconPath:          strings.TrimSpace(r.FormValue("icon_path")),
	}
	if app.ServerMode == "" {
		app.ServerMode = "ssh"
	}
	if app.ServerPort == "" {
		app.ServerPort = "3000"
	}
	if app.SSHPort == "" {
		app.SSHPort = "22"
	}
	if app.SSHRemoteDir == "" {
		app.SSHRemoteDir = "~/ppatcher-server"
	}
	if app.ColorPalette == "" {
		app.ColorPalette = "blue"
	}
	if app.Version == "" {
		app.Version = "1.0.0"
	}
	return app
}

func validateApp(app *Application, partial bool) map[string]string {
	errors := make(map[string]string)
	if app.Name == "" {
		errors["name"] = "Application name is required"
	}
	if !partial {
		if app.ServerHost == "" {
			errors["server_host"] = "Server host is required"
		}
		if app.ServerUser == "" {
			errors["server_user"] = "SSH user is required"
		}
	}
	return errors
}

func renderTemplate(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("Template render error (%s): %v", name, err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
	}
}
