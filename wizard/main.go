package main

import (
	_ "embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

//go:embed static/index.html
var indexHTML []byte

var (
	ppatcherRoot string
	state        = &WizardState{}
	mu           sync.Mutex
	serverProc   *exec.Cmd
	buildLog     strings.Builder
	buildDone    bool
	buildErr     string
)

type WizardState struct {
	ProjectName  string   `json:"projectName"`
	ProjectDir   string   `json:"projectDir"`
	DisplayName  string   `json:"displayName"`
	Description  string   `json:"description"`
	Version      string   `json:"version"`
	Title        string   `json:"title"`
	ColorPalette string   `json:"colorPalette"`
	Executable   string   `json:"executable"`
	OutputName   string   `json:"outputName"`
	Logo         string   `json:"logo"`
	Icon         string   `json:"icon"`
	Backend      string   `json:"backend"`
	Port         string   `json:"port"`
	ServerMode   string   `json:"serverMode"`
	FilesDir     string   `json:"filesDir"`
	SSHHost      string   `json:"sshHost"`
	SSHUser      string   `json:"sshUser"`
	SSHPort      string   `json:"sshPort"`
	SSHKeyPath   string   `json:"sshKeyPath"`
	SSHPassword  string   `json:"sshPassword"`
	SSHRemoteDir string   `json:"sshRemoteDir"`
	Platforms    []string `json:"platforms"`
}

type APIResponse struct {
	OK    bool        `json:"ok"`
	Data  interface{} `json:"data,omitempty"`
	Error string      `json:"error,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(indexHTML)
}

func handleGetState(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	writeJSON(w, 200, APIResponse{OK: true, Data: state})
}

func handleCreateProject(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, 400, APIResponse{Error: "Invalid request"})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeJSON(w, 400, APIResponse{Error: "Project name is required"})
		return
	}
	// Reject path separators and dots to prevent traversal
	if strings.ContainsAny(name, "/\\..") {
		writeJSON(w, 400, APIResponse{Error: "Project name must not contain path separators or dots"})
		return
	}

	projectDir := filepath.Join(ppatcherRoot, name)
	if _, err := os.Stat(projectDir); err == nil {
		writeJSON(w, 400, APIResponse{Error: "Directory already exists: " + name})
		return
	}

	if err := os.MkdirAll(filepath.Join(projectDir, "files"), 0755); err != nil {
		writeJSON(w, 500, APIResponse{Error: "Failed to create directory: " + err.Error()})
		return
	}

	mu.Lock()
	state.ProjectName = name
	state.ProjectDir = projectDir
	state.FilesDir = filepath.Join(projectDir, "files")
	state.Port = "3000"
	state.ServerMode = "local"
	state.ColorPalette = "neutral"
	state.Version = "1.0.0"
	state.Description = "Keep your files up to date"
	state.Title = name + " Patcher"
	state.DisplayName = name
	state.OutputName = name + "-patcher"
	state.Backend = "http://localhost:3000"
	mu.Unlock()

	saveConfig()

	writeJSON(w, 200, APIResponse{OK: true, Data: map[string]string{
		"dir": projectDir,
	}})
}

func handleSaveBranding(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DisplayName  string `json:"displayName"`
		Description  string `json:"description"`
		Version      string `json:"version"`
		Title        string `json:"title"`
		ColorPalette string `json:"colorPalette"`
		Executable   string `json:"executable"`
		OutputName   string `json:"outputName"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, 400, APIResponse{Error: "Invalid request"})
		return
	}

	mu.Lock()
	if req.DisplayName != "" {
		state.DisplayName = req.DisplayName
	}
	state.Description = req.Description
	if req.Version != "" {
		state.Version = req.Version
	}
	if req.Title != "" {
		state.Title = req.Title
	}
	if req.ColorPalette != "" {
		state.ColorPalette = req.ColorPalette
	}
	state.Executable = req.Executable
	if req.OutputName != "" {
		state.OutputName = req.OutputName
	}
	mu.Unlock()

	if err := saveConfig(); err != nil {
		writeJSON(w, 500, APIResponse{Error: err.Error()})
		return
	}

	writeJSON(w, 200, APIResponse{OK: true})
}

func handleUpload(w http.ResponseWriter, r *http.Request, field string) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeJSON(w, 400, APIResponse{Error: "File too large (max 10MB)"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, APIResponse{Error: "No file uploaded"})
		return
	}
	defer file.Close()

	mu.Lock()
	projectDir := state.ProjectDir
	mu.Unlock()

	if projectDir == "" {
		writeJSON(w, 400, APIResponse{Error: "Create project first"})
		return
	}

	ext := filepath.Ext(header.Filename)
	destPath := filepath.Join(projectDir, field+ext)
	out, err := os.Create(destPath)
	if err != nil {
		writeJSON(w, 500, APIResponse{Error: err.Error()})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		writeJSON(w, 500, APIResponse{Error: err.Error()})
		return
	}

	mu.Lock()
	if field == "logo" {
		state.Logo = destPath
	} else {
		state.Icon = destPath
	}
	mu.Unlock()

	saveConfig()

	writeJSON(w, 200, APIResponse{OK: true, Data: map[string]string{
		"path": destPath,
		"name": header.Filename,
	}})
}

func handlePreviewFile(w http.ResponseWriter, r *http.Request) {
	fileType := r.PathValue("type")
	mu.Lock()
	var path string
	if fileType == "logo" {
		path = state.Logo
	} else if fileType == "icon" {
		path = state.Icon
	}
	mu.Unlock()

	if path == "" {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, path)
}

func handleServerSetup(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode     string `json:"mode"`
		Port     string `json:"port"`
		FilesDir string `json:"filesDir"`
		SSHHost  string `json:"sshHost"`
		SSHUser  string `json:"sshUser"`
		SSHPort  string `json:"sshPort"`
		SSHKey   string `json:"sshKeyPath"`
		SSHPass  string `json:"sshPassword"`
		SSHDir   string `json:"sshRemoteDir"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, 400, APIResponse{Error: "Invalid request"})
		return
	}

	mu.Lock()
	state.ServerMode = req.Mode
	if req.Port != "" {
		state.Port = req.Port
	}
	if req.FilesDir != "" {
		state.FilesDir = req.FilesDir
	}
	state.SSHHost = req.SSHHost
	state.SSHUser = req.SSHUser
	state.SSHPort = req.SSHPort
	state.SSHKeyPath = req.SSHKey
	state.SSHPassword = req.SSHPass
	state.SSHRemoteDir = req.SSHDir

	if req.Mode == "local" {
		state.Backend = fmt.Sprintf("http://localhost:%s", state.Port)
	} else if req.SSHHost != "" {
		state.Backend = fmt.Sprintf("http://%s:%s", req.SSHHost, state.Port)
	}
	mu.Unlock()

	saveConfig()
	writeJSON(w, 200, APIResponse{OK: true})
}

func handleStartLocalServer(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	port := state.Port
	filesDir := state.FilesDir
	projectDir := state.ProjectDir
	mu.Unlock()

	if port == "" {
		port = "3000"
	}

	stopServer()

	// Build server binary into project directory
	serverBin := filepath.Join(projectDir, "fileserver")
	buildCmd := exec.Command("go", "build", "-o", serverBin, "./server/")
	buildCmd.Dir = ppatcherRoot
	if out, err := buildCmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, APIResponse{Error: fmt.Sprintf("Failed to build server: %s\n%s", err, string(out))})
		return
	}

	cmd := exec.Command(serverBin)
	cmd.Dir = projectDir
	cmd.Env = append(os.Environ(),
		"PORT=:"+port,
		"FILES_DIR="+filesDir,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		writeJSON(w, 500, APIResponse{Error: "Failed to start server: " + err.Error()})
		return
	}

	mu.Lock()
	serverProc = cmd
	mu.Unlock()

	// Wait for server to start
	time.Sleep(2 * time.Second)

	writeJSON(w, 200, APIResponse{OK: true, Data: map[string]string{
		"url": fmt.Sprintf("http://localhost:%s", port),
	}})
}

func handleStopServer(w http.ResponseWriter, r *http.Request) {
	stopServer()
	writeJSON(w, 200, APIResponse{OK: true})
}

func handleUploadSSHKey(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	projectDir := state.ProjectDir
	mu.Unlock()

	if projectDir == "" {
		writeJSON(w, 400, APIResponse{Error: "Create project first"})
		return
	}

	contentType := r.Header.Get("Content-Type")
	keyPath := filepath.Join(projectDir, "ssh_key")

	if strings.HasPrefix(contentType, "multipart/form-data") {
		// File upload
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			writeJSON(w, 400, APIResponse{Error: "File too large (max 1MB)"})
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			writeJSON(w, 400, APIResponse{Error: "No file uploaded"})
			return
		}
		defer file.Close()
		content, err := io.ReadAll(io.LimitReader(file, 1<<20))
		if err != nil {
			writeJSON(w, 500, APIResponse{Error: err.Error()})
			return
		}
		if err := os.WriteFile(keyPath, content, 0600); err != nil {
			writeJSON(w, 500, APIResponse{Error: err.Error()})
			return
		}
	} else {
		// JSON with pasted content
		var req struct {
			Content string `json:"content"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
			writeJSON(w, 400, APIResponse{Error: "Invalid request"})
			return
		}
		content := strings.TrimSpace(req.Content)
		if content == "" {
			writeJSON(w, 400, APIResponse{Error: "Key content is empty"})
			return
		}
		if err := os.WriteFile(keyPath, []byte(content+"\n"), 0600); err != nil {
			writeJSON(w, 500, APIResponse{Error: err.Error()})
			return
		}
	}

	mu.Lock()
	state.SSHKeyPath = keyPath
	mu.Unlock()

	writeJSON(w, 200, APIResponse{OK: true, Data: map[string]string{"path": keyPath}})
}

func handleTestSSH(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	host := state.SSHHost
	user := state.SSHUser
	port := state.SSHPort
	keyPath := state.SSHKeyPath
	password := state.SSHPassword
	mu.Unlock()

	if host == "" || user == "" {
		writeJSON(w, 400, APIResponse{Error: "SSH host and user are required"})
		return
	}
	if port == "" {
		port = "22"
	}

	cmd := buildSSHCmd(host, user, port, keyPath, password, "echo", "ok")
	output, err := cmd.CombinedOutput()
	if err != nil {
		writeJSON(w, 200, APIResponse{OK: false, Error: fmt.Sprintf("SSH connection failed: %s\n%s", err, strings.TrimSpace(string(output)))})
		return
	}

	writeJSON(w, 200, APIResponse{OK: true, Data: map[string]string{
		"output": strings.TrimSpace(string(output)),
	}})
}

func handleDeploySSH(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	host := state.SSHHost
	user := state.SSHUser
	port := state.SSHPort
	keyPath := state.SSHKeyPath
	password := state.SSHPassword
	remoteDir := state.SSHRemoteDir
	filesDir := state.FilesDir
	serverPort := state.Port
	mu.Unlock()

	if remoteDir == "" {
		remoteDir = "~/ppatcher-server"
	}
	if port == "" {
		port = "22"
	}
	if serverPort == "" {
		serverPort = "3000"
	}

	target := fmt.Sprintf("%s@%s", user, host)

	// 1. Create remote directory
	cmd := buildSSHCmd(host, user, port, keyPath, password, "mkdir", "-p", remoteDir+"/files")
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, APIResponse{Error: fmt.Sprintf("Failed to create remote directory: %s\n%s", err, out)})
		return
	}

	// 2. Build server binary for linux/amd64
	serverBin := filepath.Join(ppatcherRoot, "build", "bin", "fileserver-deploy")
	os.MkdirAll(filepath.Dir(serverBin), 0755)
	buildCmd := exec.Command("go", "build", "-o", serverBin, "./server/")
	buildCmd.Dir = ppatcherRoot
	buildCmd.Env = append(os.Environ(), "GOOS=linux", "GOARCH=amd64", "CGO_ENABLED=0")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, APIResponse{Error: fmt.Sprintf("Failed to build server: %s\n%s", err, out)})
		return
	}

	// 3. Copy server binary
	cmd = buildSCPCmd(port, keyPath, password, serverBin, target+":"+remoteDir+"/fileserver")
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, APIResponse{Error: fmt.Sprintf("Failed to upload server binary: %s\n%s", err, out)})
		return
	}

	// 4. Copy files directory
	cmd = buildSCPCmd(port, keyPath, password, "-r", filesDir+"/.", target+":"+remoteDir+"/files/")
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, APIResponse{Error: fmt.Sprintf("Failed to upload files: %s\n%s", err, out)})
		return
	}

	// 5. Start server remotely
	startScript := fmt.Sprintf(
		"cd %s && chmod +x fileserver && (pkill -f './fileserver' 2>/dev/null || true) && sleep 1 && PORT=:%s FILES_DIR=./files nohup ./fileserver > server.log 2>&1 &",
		remoteDir, serverPort,
	)
	cmd = buildSSHCmd(host, user, port, keyPath, password, startScript)
	if out, err := cmd.CombinedOutput(); err != nil {
		writeJSON(w, 500, APIResponse{Error: fmt.Sprintf("Failed to start remote server: %s\n%s", err, out)})
		return
	}

	writeJSON(w, 200, APIResponse{OK: true})
}

func handleTestConnection(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	backend := state.Backend
	mu.Unlock()

	if backend == "" {
		writeJSON(w, 400, APIResponse{Error: "Backend URL not configured"})
		return
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(backend + "/meta")
	if err != nil {
		writeJSON(w, 200, APIResponse{OK: false, Error: "Cannot connect to server: " + err.Error()})
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))

	if resp.StatusCode != 200 {
		writeJSON(w, 200, APIResponse{OK: false, Error: fmt.Sprintf("Server returned HTTP %d", resp.StatusCode)})
		return
	}

	// Parse meta to show useful info
	var meta struct {
		Hash      string `json:"hash"`
		TotalSize int64  `json:"totalSize"`
	}
	json.Unmarshal(body, &meta)

	writeJSON(w, 200, APIResponse{OK: true, Data: map[string]interface{}{
		"hash":      meta.Hash,
		"totalSize": meta.TotalSize,
		"raw":       string(body),
	}})
}

func handleBuild(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Platforms []string `json:"platforms"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, 400, APIResponse{Error: "Invalid request"})
		return
	}

	if len(req.Platforms) == 0 {
		writeJSON(w, 400, APIResponse{Error: "Select at least one platform"})
		return
	}

	mu.Lock()
	state.Platforms = req.Platforms
	buildLog.Reset()
	buildDone = false
	buildErr = ""
	projectDir := state.ProjectDir
	outputName := state.OutputName
	mu.Unlock()

	saveConfig()

	// Copy config to ppatcher root for the build
	configSrc := filepath.Join(projectDir, "config.json")
	configDst := filepath.Join(ppatcherRoot, "config.json")
	data, err := os.ReadFile(configSrc)
	if err != nil {
		writeJSON(w, 500, APIResponse{Error: "Cannot read config: " + err.Error()})
		return
	}
	if err := os.WriteFile(configDst, data, 0644); err != nil {
		writeJSON(w, 500, APIResponse{Error: "Cannot write config: " + err.Error()})
		return
	}

	platforms := strings.Join(req.Platforms, ",")

	go func() {
		cmd := exec.Command("./build-client.sh",
			"--config", "config.json",
			"--platforms", platforms,
		)
		cmd.Dir = ppatcherRoot
		output, err := cmd.CombinedOutput()

		mu.Lock()
		buildLog.Write(output)
		buildDone = true
		if err != nil {
			buildErr = err.Error()
		}
		mu.Unlock()

		// Copy built files to project dist/
		distDir := filepath.Join(projectDir, "dist")
		os.MkdirAll(distDir, 0755)
		if outputName == "" {
			outputName = "ppatcher"
		}
		matches, _ := filepath.Glob(filepath.Join(ppatcherRoot, "build", "bin", outputName+"*"))
		for _, m := range matches {
			fdata, ferr := os.ReadFile(m)
			if ferr == nil {
				os.WriteFile(filepath.Join(distDir, filepath.Base(m)), fdata, 0755)
			}
		}
	}()

	writeJSON(w, 200, APIResponse{OK: true})
}

func handleBuildStatus(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	logStr := buildLog.String()
	done := buildDone
	errStr := buildErr
	projectDir := state.ProjectDir
	mu.Unlock()

	var files []map[string]interface{}
	if done && errStr == "" {
		distDir := filepath.Join(projectDir, "dist")
		entries, _ := os.ReadDir(distDir)
		for _, e := range entries {
			if !e.IsDir() {
				info, _ := e.Info()
				size := int64(0)
				if info != nil {
					size = info.Size()
				}
				files = append(files, map[string]interface{}{
					"name": e.Name(),
					"size": size,
				})
			}
		}
	}

	writeJSON(w, 200, APIResponse{OK: true, Data: map[string]interface{}{
		"log":   logStr,
		"done":  done,
		"error": errStr,
		"files": files,
	}})
}

func saveConfig() error {
	mu.Lock()
	defer mu.Unlock()

	if state.ProjectDir == "" {
		return fmt.Errorf("no project directory")
	}

	config := map[string]interface{}{
		"backend":      state.Backend,
		"executable":   state.Executable,
		"colorPalette": state.ColorPalette,
		"mode":         "production",
		"version":      state.Version,
		"description":  state.Description,
		"title":        state.Title,
		"displayName":  state.DisplayName,
		"outputName":   state.OutputName,
		"logo":         state.Logo,
		"icon":         state.Icon,
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(state.ProjectDir, "config.json"), data, 0644)
}

func stopServer() {
	mu.Lock()
	proc := serverProc
	serverProc = nil
	mu.Unlock()

	if proc != nil && proc.Process != nil {
		proc.Process.Kill()
		proc.Wait()
	}
}

// buildSSHCmd creates an ssh exec.Cmd, using sshpass for password auth when needed.
// remoteArgs are appended after user@host (e.g. the remote command to run).
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
		// Use sshpass for password-based auth
		return exec.Command("sshpass", append([]string{"-p", password, "ssh"}, sshArgs...)...)
	}
	return exec.Command("ssh", sshArgs...)
}

// buildSCPCmd creates a scp exec.Cmd, using sshpass for password auth when needed.
// scpArgs are the scp-specific args (e.g. "-r", src, dest).
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

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start()
}

func findPPatcherRoot() string {
	// 1. Check --root flag (handled by caller)
	// 2. Check current working directory
	cwd, _ := os.Getwd()
	if _, err := os.Stat(filepath.Join(cwd, "build-client.sh")); err == nil {
		return cwd
	}
	// 3. Check executable's parent directory (for wizard/ subdirectory)
	if exe, err := os.Executable(); err == nil {
		parent := filepath.Dir(filepath.Dir(exe))
		if _, err := os.Stat(filepath.Join(parent, "build-client.sh")); err == nil {
			return parent
		}
	}
	return cwd
}

func main() {
	portFlag := flag.String("port", "0", "Port for wizard UI (0 = random)")
	rootFlag := flag.String("root", "", "PPatcher project root directory")
	noBrowser := flag.Bool("no-browser", false, "Don't open browser automatically")
	flag.Parse()

	if *rootFlag != "" {
		ppatcherRoot = *rootFlag
	} else {
		ppatcherRoot = findPPatcherRoot()
	}

	log.Printf("PPatcher root: %s", ppatcherRoot)

	// Signal handling for cleanup
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		stopServer()
		os.Exit(0)
	}()

	// Routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", handleIndex)
	mux.HandleFunc("GET /api/state", handleGetState)
	mux.HandleFunc("POST /api/project", handleCreateProject)
	mux.HandleFunc("PUT /api/branding", handleSaveBranding)
	mux.HandleFunc("POST /api/upload/logo", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, "logo")
	})
	mux.HandleFunc("POST /api/upload/icon", func(w http.ResponseWriter, r *http.Request) {
		handleUpload(w, r, "icon")
	})
	mux.HandleFunc("GET /api/preview/{type}", handlePreviewFile)
	mux.HandleFunc("POST /api/server/setup", handleServerSetup)
	mux.HandleFunc("POST /api/server/start", handleStartLocalServer)
	mux.HandleFunc("POST /api/server/stop", handleStopServer)
	mux.HandleFunc("POST /api/upload/sshkey", handleUploadSSHKey)
	mux.HandleFunc("POST /api/server/test-ssh", handleTestSSH)
	mux.HandleFunc("POST /api/server/deploy", handleDeploySSH)
	mux.HandleFunc("POST /api/test", handleTestConnection)
	mux.HandleFunc("POST /api/build", handleBuild)
	mux.HandleFunc("GET /api/build/status", handleBuildStatus)

	// Find available port
	port := *portFlag
	if port == "0" {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			log.Fatal(err)
		}
		port = fmt.Sprintf("%d", ln.Addr().(*net.TCPAddr).Port)
		ln.Close()
	}

	url := fmt.Sprintf("http://localhost:%s", port)
	log.Printf("Setup wizard: %s", url)

	if !*noBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			openBrowser(url)
		}()
	}

	server := &http.Server{
		Addr:              "127.0.0.1:" + port,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Fatal(server.ListenAndServe())
}
