package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Supported targets for server binary cross-compilation.
var serverTargets = []struct {
	GOOS, GOARCH string
}{
	{"linux", "amd64"},
	{"linux", "arm64"},
	{"darwin", "amd64"},
	{"darwin", "arm64"},
}

// serverBinsDir returns the directory where pre-built server binaries are stored.
func serverBinsDir() string {
	return filepath.Join(ppatcherRoot(), "build", "server-bins")
}

// serverBinPath returns the path for a specific OS/arch binary.
func serverBinPath(goos, goarch string) string {
	return filepath.Join(serverBinsDir(), fmt.Sprintf("fileserver-%s-%s", goos, goarch))
}

// ensureServerBins builds server binaries for all supported targets if they don't exist.
func ensureServerBins() error {
	root := ppatcherRoot()
	dir := serverBinsDir()
	os.MkdirAll(dir, 0755)

	for _, t := range serverTargets {
		bin := serverBinPath(t.GOOS, t.GOARCH)
		if _, err := os.Stat(bin); err == nil {
			continue // already built
		}
		log.Printf("[server-bins] building %s/%s ...", t.GOOS, t.GOARCH)
		cmd := exec.Command("go", "build", "-o", bin, "./server/")
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GOOS="+t.GOOS, "GOARCH="+t.GOARCH, "CGO_ENABLED=0")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("build %s/%s: %s\n%s", t.GOOS, t.GOARCH, err, out)
		}
		log.Printf("[server-bins] built %s", bin)
	}
	return nil
}

// rebuildServerBins forces a rebuild of all server binaries.
func rebuildServerBins() error {
	dir := serverBinsDir()
	os.RemoveAll(dir)
	return ensureServerBins()
}

// detectRemoteTarget uses SSH to determine the remote server's OS and architecture.
func detectRemoteTarget(host, user, port, keyPath, password string) (goos, goarch string, err error) {
	cmd := buildSSHCmd(host, user, port, keyPath, password, "uname -s -m")
	out, err := cmd.Output() // Output() captures only stdout, ignoring SSH warnings on stderr
	if err != nil {
		return "", "", fmt.Errorf("failed to detect remote OS: %s\n%s", err, out)
	}
	parts := strings.Fields(strings.TrimSpace(string(out)))
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected uname output: %s", out)
	}

	// Map uname output to GOOS/GOARCH
	switch strings.ToLower(parts[0]) {
	case "linux":
		goos = "linux"
	case "darwin":
		goos = "darwin"
	default:
		return "", "", fmt.Errorf("unsupported OS: %s", parts[0])
	}

	switch parts[1] {
	case "x86_64", "amd64":
		goarch = "amd64"
	case "aarch64", "arm64":
		goarch = "arm64"
	default:
		return "", "", fmt.Errorf("unsupported architecture: %s", parts[1])
	}

	return goos, goarch, nil
}

// handleRebuildServerBins forces a rebuild of all server binaries.
func handleRebuildServerBins(w http.ResponseWriter, r *http.Request) {
	if err := rebuildServerBins(); err != nil {
		writeJSON(w, 500, apiResponse{Error: fmt.Sprintf("Failed to rebuild: %s", err)})
		return
	}
	writeJSON(w, 200, apiResponse{OK: true, Data: "All server binaries rebuilt"})
}

type apiResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, resp apiResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

// ---------- Test Connection ----------

// ---------- Verify Backend ----------

func handleVerifyBackend(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	backend := app.BackendURL
	if backend == "" {
		writeJSON(w, 400, apiResponse{Error: "Backend URL not configured"})
		return
	}

	expectedKey := r.FormValue("deploy_key")
	if expectedKey == "" {
		writeJSON(w, 400, apiResponse{Error: "Deploy key is required"})
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(backend + "/health")
	if err != nil {
		writeJSON(w, 200, apiResponse{OK: false, Error: "Cannot connect to server: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != 200 {
		writeJSON(w, 200, apiResponse{OK: false, Error: fmt.Sprintf("Server returned HTTP %d", resp.StatusCode)})
		return
	}

	var health struct {
		Status    string `json:"status"`
		DeployKey string `json:"deploy_key"`
	}
	if err := json.Unmarshal(body, &health); err != nil {
		writeJSON(w, 200, apiResponse{OK: false, Error: "Invalid health response"})
		return
	}

	if health.DeployKey != expectedKey {
		writeJSON(w, 200, apiResponse{OK: false, Error: "Deploy key mismatch — the server may not have restarted correctly"})
		return
	}

	writeJSON(w, 200, apiResponse{OK: true, Data: "Backend verified"})
}

// ---------- Test Connection ----------

func handleTestConnection(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	backend := app.BackendURL
	if backend == "" {
		writeJSON(w, 400, apiResponse{Error: "Backend URL not configured"})
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(backend + "/health")
	if err != nil {
		writeJSON(w, 200, apiResponse{OK: false, Error: "Cannot connect to server: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != 200 {
		writeJSON(w, 200, apiResponse{OK: false, Error: fmt.Sprintf("Server returned HTTP %d", resp.StatusCode)})
		return
	}

	var meta struct {
		Hash      string `json:"hash"`
		TotalSize int64  `json:"totalSize"`
	}
	json.Unmarshal(body, &meta)

	writeJSON(w, 200, apiResponse{OK: true, Data: map[string]interface{}{
		"hash":      meta.Hash,
		"totalSize": meta.TotalSize,
		"raw":       string(body),
	}})
}

// ---------- Test SSH ----------

func handleTestSSH(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if app.ServerHost == "" || app.ServerUser == "" {
		writeJSON(w, 400, apiResponse{Error: "SSH host and user are required"})
		return
	}

	sshPort := app.SSHPort
	if sshPort == "" {
		sshPort = "22"
	}

	cmd := buildSSHCmd(app.ServerHost, app.ServerUser, sshPort, app.SSHKeyPath, app.SSHPassword, "echo", "ok")
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, 200, apiResponse{OK: false, Error: fmt.Sprintf("SSH connection failed: %s\n%s", err, strings.TrimSpace(string(output)))})
		return
	}

	writeJSON(w, 200, apiResponse{OK: true, Data: map[string]string{
		"output": strings.TrimSpace(string(output)),
	}})
}

// ---------- Deploy SSH ----------

func handleDeploySSH(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	host := app.ServerHost
	sshUser := app.ServerUser
	sshPort := app.SSHPort
	keyPath := app.SSHKeyPath
	password := app.SSHPassword
	remoteDir := app.SSHRemoteDir
	serverPort := app.ServerPort

	if host == "" || sshUser == "" {
		writeJSON(w, 400, apiResponse{Error: "SSH host and user are required"})
		return
	}
	if remoteDir == "" {
		remoteDir = "~/ppatcher-server"
	}
	if sshPort == "" {
		sshPort = "22"
	}
	if serverPort == "" {
		serverPort = "3000"
	}

	target := fmt.Sprintf("%s@%s", sshUser, host)

	// 1. Detect remote OS and architecture
	goos, goarch, err := detectRemoteTarget(host, sshUser, sshPort, keyPath, password)
	if err != nil {
		writeJSON(w, 500, apiResponse{Error: err.Error()})
		return
	}
	log.Printf("[deploy] detected remote target: %s/%s", goos, goarch)

	// 2. Ensure pre-built binaries exist
	if err := ensureServerBins(); err != nil {
		writeJSON(w, 500, apiResponse{Error: fmt.Sprintf("Failed to build server binaries: %s", err)})
		return
	}

	serverBin := serverBinPath(goos, goarch)
	if _, err := os.Stat(serverBin); err != nil {
		writeJSON(w, 500, apiResponse{Error: fmt.Sprintf("No server binary for %s/%s", goos, goarch)})
		return
	}

	// 3. Create remote directory, stop any running server, install unzip/unrar, and open firewall port
	setupScript := fmt.Sprintf(
		"mkdir -p %s/files && pkill fileserver 2>/dev/null; sleep 1; "+
			"(command -v unzip >/dev/null 2>&1 || ((sudo -n true >/dev/null 2>&1 && sudo apt-get update -qq && sudo apt-get install -y -qq unzip unrar-free) || (apt-get update -qq && apt-get install -y -qq unzip unrar-free) || (sudo -n yum install -y -q unzip unrar || yum install -y -q unzip unrar) || (sudo -n apk add --no-cache unzip unrar || apk add --no-cache unzip unrar) || (brew install unzip unrar) || true)); "+
			"(command -v unrar >/dev/null 2>&1 || true); "+
			"(sudo -n ufw allow %s/tcp 2>/dev/null || sudo -n firewall-cmd --permanent --add-port=%s/tcp && sudo -n firewall-cmd --reload 2>/dev/null || sudo -n iptables -I INPUT -p tcp --dport %s -j ACCEPT 2>/dev/null || true); true",
		remoteDir, serverPort, serverPort, serverPort,
	)
	cmd := buildSSHCmd(host, sshUser, sshPort, keyPath, password, setupScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("[deploy] setup output: %s", strings.TrimSpace(string(out)))
		writeJSON(w, 500, apiResponse{Error: fmt.Sprintf("Failed to prepare remote directory: %s\n%s", err, out)})
		return
	}

	// 4. Copy server binary
	cmd = buildSCPCmd(sshPort, keyPath, password, serverBin, target+":"+remoteDir+"/fileserver")
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, apiResponse{Error: fmt.Sprintf("Failed to upload server binary: %s\n%s", err, out)})
		return
	}

	// 5. Copy files directory if it exists
	filesDir := appFilesDir(app)
	if info, ferr := os.Stat(filesDir); ferr == nil && info.IsDir() {
		cmd = buildSCPCmd(sshPort, keyPath, password, "-r", filesDir+"/.", target+":"+remoteDir+"/files/")
		if out, err := cmd.CombinedOutput(); err != nil {
			writeJSON(w, 500, apiResponse{Error: fmt.Sprintf("Failed to upload files: %s\n%s", err, out)})
			return
		}
	}

	// 6. Generate a deploy key and admin key, then start server remotely
	deployKey := uuid.New().String()
	adminKey := uuid.New().String()
	// Wrap in a subshell so SSH doesn't wait for the background process.
	// setsid detaches from the session so the process survives SSH disconnect.
	startScript := fmt.Sprintf(
		"cd %s && chmod +x fileserver && (PORT=:%s FILES_DIR=./files DEPLOY_KEY=%s ADMIN_KEY=%s setsid ./fileserver > server.log 2>&1 < /dev/null &)",
		remoteDir, serverPort, deployKey, adminKey,
	)
	log.Printf("[deploy] start script: %s", startScript)
	cmd = buildSSHCmd(host, sshUser, sshPort, keyPath, password, startScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, apiResponse{Error: fmt.Sprintf("Failed to start remote server: %s\n%s", err, out)})
		return
	}

	// 7. Verify the server actually started
	time.Sleep(3 * time.Second)
	verifyScript := "pgrep -x fileserver > /dev/null 2>&1 && echo OK || echo FAIL"
	log.Printf("[deploy] verify script: %s", verifyScript)
	cmd = buildSSHCmd(host, sshUser, sshPort, keyPath, password, verifyScript)
	verifyOut, _ := cmd.CombinedOutput()
	log.Printf("[deploy] verify output: %s", strings.TrimSpace(string(verifyOut)))
	if !strings.Contains(string(verifyOut), "OK") {
		// Try to get the server log for debugging
		logScript := fmt.Sprintf("cat %s/server.log 2>/dev/null | tail -20", remoteDir)
		cmd = buildSSHCmd(host, sshUser, sshPort, keyPath, password, logScript)
		logOut, _ := cmd.CombinedOutput()
		log.Printf("[deploy] server.log: %s", strings.TrimSpace(string(logOut)))
		errMsg := "Server binary was deployed but failed to start."
		if len(logOut) > 0 {
			errMsg += "\nServer log:\n" + strings.TrimSpace(string(logOut))
		}
		writeJSON(w, 500, apiResponse{Error: errMsg})
		return
	}

	// Update the backend URL and admin key.
	// Use context.Background() so a dropped HTTP connection can't leave
	// the DB and the live server with mismatched keys.
	backendURL := fmt.Sprintf("http://%s:%s", host, serverPort)
	if _, err := db.Exec(context.Background(),
		`UPDATE applications SET backend_url=$1, admin_key=$2, updated_at=$3 WHERE id=$4 AND user_id=$5`,
		backendURL, adminKey, time.Now(), app.ID, user.ID,
	); err != nil {
		log.Printf("[deploy] failed to update DB after deploy: %v", err)
	}

	writeJSON(w, 200, apiResponse{OK: true, Data: map[string]string{
		"url":        backendURL,
		"deploy_key": deployKey,
		"admin_key":  adminKey,
	}})
}

// ---------- SSH Helpers ----------

func buildSSHCmd(host, user, port, keyPath, password string, remoteArgs ...string) *exec.Cmd {
	sshArgs := []string{
		"-o", "ConnectTimeout=5",
		"-o", "StrictHostKeyChecking=accept-new",
		"-p", port,
	}
	if keyPath != "" {
		sshArgs = append(sshArgs, "-i", keyPath)
	}
	sshArgs = append(sshArgs, fmt.Sprintf("%s@%s", user, host))
	sshArgs = append(sshArgs, remoteArgs...)

	if password != "" && keyPath == "" {
		return exec.Command("sshpass", append([]string{"-p", password, "ssh"}, sshArgs...)...)
	}
	return exec.Command("ssh", sshArgs...)
}

func buildSCPCmd(port, keyPath, password string, scpArgs ...string) *exec.Cmd {
	baseArgs := []string{
		"-o", "StrictHostKeyChecking=accept-new",
		"-P", port,
	}
	if keyPath != "" {
		baseArgs = append(baseArgs, "-i", keyPath)
	}
	allArgs := append(baseArgs, scpArgs...)

	if password != "" && keyPath == "" {
		return exec.Command("sshpass", append([]string{"-p", password, "scp"}, allArgs...)...)
	}
	return exec.Command("scp", allArgs...)
}

// appFilesDir returns the files directory for the app.
func appFilesDir(app *Application) string {
	if app.FilesDir != "" {
		return app.FilesDir
	}
	return filepath.Join(appDistDir(app.ID), "files")
}

// ---------- Get Admin Key ----------

// handleGetAdminKey returns the stored admin key for the app so the frontend
// can authenticate directly with the fileserver.
func handleGetAdminKey(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	if app.AdminKey == "" {
		writeJSON(w, 200, apiResponse{OK: false, Error: "No admin key configured. Deploy the server first."})
		return
	}

	writeJSON(w, 200, apiResponse{OK: true, Data: map[string]string{
		"admin_key":   app.AdminKey,
		"backend_url": app.BackendURL,
	}})
}

// handleUploadSSHKey accepts an SSH key via file upload or pasted content.
func handleUploadSSHKey(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	distDir := appDistDir(app.ID)
	os.MkdirAll(distDir, 0755)
	keyPath := filepath.Join(distDir, "ssh_key")

	var rawContent []byte
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			writeJSON(w, 400, apiResponse{Error: "File too large (max 1MB)"})
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, 400, apiResponse{Error: "No file uploaded"})
			return
		}
		defer file.Close()
		rawContent, err = io.ReadAll(io.LimitReader(file, 1<<20))
		if err != nil {
			writeJSON(w, 500, apiResponse{Error: err.Error()})
			return
		}
	} else {
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeJSON(w, 400, apiResponse{Error: "Invalid request"})
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			writeJSON(w, 400, apiResponse{Error: "Key content is empty"})
			return
		}
		rawContent = []byte(req.Content)
	}

	// Normalize the key: strip BOM, convert CRLF/CR to LF, trim trailing whitespace, ensure trailing newline
	normalized := string(rawContent)
	normalized = strings.TrimPrefix(normalized, "\xef\xbb\xbf") // UTF-8 BOM
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")   // CRLF -> LF
	normalized = strings.ReplaceAll(normalized, "\r", "\n")     // lone CR -> LF
	normalized = strings.TrimSpace(normalized) + "\n"

	if err := os.WriteFile(keyPath, []byte(normalized), 0600); err != nil {
		writeJSON(w, 500, apiResponse{Error: err.Error()})
		return
	}

	// Save key path to DB
	_, _ = db.Exec(r.Context(),
		`UPDATE applications SET ssh_key_path=$1, updated_at=$2 WHERE id=$3 AND user_id=$4`,
		keyPath, time.Now(), app.ID, user.ID,
	)

	log.Printf("SSH key saved for app %d", app.ID)
	writeJSON(w, 200, apiResponse{OK: true})
}
