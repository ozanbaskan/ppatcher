package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

type MetaData struct {
	Hash      string `json:"hash"`
	TotalSize int64  `json:"totalSize"`
}

type MetaDataForFiles struct {
	Files []MetaForFile `json:"files"`
}

type MetaForFile struct {
	Hash string `json:"hash"`
	Path string `json:"path"`
	Size int64  `json:"size"`
}

var (
	metaFile     = "meta.json"
	filesmetaFile = "filesmeta.json"
	versionFile   = "version.txt"
	adminKeyFile  = "adminkey.txt"
)

var (
	port            = ":3000"
	filesDir        = "./files"
	adminKey        string
	metaCache       []byte
	filesMetaCache  []byte
	versionCache    string
	cacheMutex      sync.RWMutex
	bufferPool     = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024) // 32KB buffers
		},
	}
)

// ---------- Rate limiter for admin endpoints ----------

type rateLimiter struct {
	mu       sync.Mutex
	attempts map[string][]time.Time
	max      int
	window   time.Duration
}

func newRateLimiter(max int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		attempts: make(map[string][]time.Time),
		max:      max,
		window:   window,
	}
}

func (rl *rateLimiter) allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Prune old entries
	valid := rl.attempts[ip][:0]
	for _, t := range rl.attempts[ip] {
		if t.After(cutoff) {
			valid = append(valid, t)
		}
	}
	rl.attempts[ip] = valid

	if len(valid) >= rl.max {
		return false
	}
	rl.attempts[ip] = append(rl.attempts[ip], now)
	return true
}

var adminLimiter = newRateLimiter(10, time.Minute) // 10 attempts per minute per IP

func main() {
	exeDir, err := os.Executable()
	if err != nil {
		log.Fatal("Could not get executable path: ", err)
	}

	exeDir = filepath.Dir(exeDir)

	err = os.Chdir(exeDir)
	if err != nil {
		log.Fatal("Could not change working directory: ", err)
	}

	filesDirEnv, exists := os.LookupEnv("FILES_DIR")
	if exists && filesDirEnv != "" {
		filesDir = filesDirEnv
	}

	port, exists = os.LookupEnv("PORT")
	if !exists || port == "" {
		port = ":3000"
	} else if port[0] != ':' {
		port = ":" + port
	}

	adminKey = os.Getenv("ADMIN_KEY")
	// If not provided via env, try to read from persisted file.
	// This keeps the key intact across restarts.
	if adminKey == "" {
		if data, err := os.ReadFile(adminKeyFile); err == nil {
			adminKey = strings.TrimSpace(string(data))
		}
	} else {
		// Persist the env-provided key so future restarts don't lose it.
		_ = os.WriteFile(adminKeyFile, []byte(adminKey), 0600)
	}

	// Create files directory if it doesn't exist
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		log.Fatalf("Failed to create files directory: %v", err)
	}

	// Generate initial meta files
	if err := generateMetaFiles(); err != nil {
		log.Fatalf("Failed to generate initial meta files: %v", err)
	}

	// Load version
	if data, err := os.ReadFile(versionFile); err == nil {
		versionCache = strings.TrimSpace(string(data))
	}
	if versionCache == "" {
		versionCache = "1.0.0"
	}

	// Start file watcher
	go watchFiles()

	// Set up HTTP handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/meta", metaHandler)
	mux.HandleFunc("/filesmeta", filesmetaHandler)
	mux.HandleFunc("/version", versionHandler)
	mux.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(filesDir))))

	// Admin endpoints (basic auth + rate limit)
	mux.HandleFunc("/admin/upload", adminAuth(adminUploadHandler))
	mux.HandleFunc("/admin/check-space", adminAuth(adminCheckSpaceHandler))
	mux.HandleFunc("/admin/version", adminAuth(adminVersionHandler))

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, withCORS(mux)))
}

func watchFiles() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Add the files directory and all subdirectories to the watcher
	err = filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := watcher.Add(path); err != nil {
				log.Printf("Failed to watch directory %s: %v", path, err)
			}
		}
		return nil
	})
	if err != nil {
		log.Fatalf("Failed to walk directory: %v", err)
	}

	log.Printf("Watching directory %s and all subdirectories for changes", filesDir)

	// Use a debounce mechanism to avoid rapid successive updates
	var debounceTimer *time.Timer
	debounceDuration := 500 * time.Millisecond

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}

			// Check if the event is for a directory
			info, err := os.Stat(event.Name)
			if err != nil {
				// If there's an error stating the file, it might have been deleted.
				// Check if the event is a remove or rename and if it's a directory we were watching.
				if event.Op&fsnotify.Remove == fsnotify.Remove || event.Op&fsnotify.Rename == fsnotify.Rename {
					// Try to remove the path from the watcher. It might be a directory.
					// Note: We don't know if it's a directory, but if we were watching it, we should remove it.
					err := watcher.Remove(event.Name)
					if err != nil {
						// It might not be a directory or we might not have been watching it, so just log.
						log.Printf("Failed to remove watcher for %s: %v", event.Name, err)
					} else {
						log.Printf("Removed watcher for deleted path: %s", event.Name)
					}
				}
				// Regenerate meta files in case of deletion
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					log.Printf("Change detected in %s, regenerating meta files", event.Name)
					if err := generateMetaFiles(); err != nil {
						log.Printf("Error generating meta files: %v", err)
					}
				})
				continue
			}

			if info.IsDir() {
				// If it's a directory and it's a create event, add it to the watcher
				if event.Op&fsnotify.Create == fsnotify.Create {
					if err := watcher.Add(event.Name); err != nil {
						log.Printf("Failed to watch new directory %s: %v", event.Name, err)
					} else {
						log.Printf("Added new directory to watcher: %s", event.Name)
					}
				}
			}

			// Only react to write, create, remove, and rename events
			if event.Op&fsnotify.Write == fsnotify.Write ||
				event.Op&fsnotify.Create == fsnotify.Create ||
				event.Op&fsnotify.Remove == fsnotify.Remove ||
				event.Op&fsnotify.Rename == fsnotify.Rename {

				// Debounce the updates
				if debounceTimer != nil {
					debounceTimer.Stop()
				}
				debounceTimer = time.AfterFunc(debounceDuration, func() {
					log.Printf("Change detected in %s, regenerating meta files", event.Name)
					if err := generateMetaFiles(); err != nil {
						log.Printf("Error generating meta files: %v", err)
					}
				})
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func generateMetaFiles() error {
	// Calculate file metadata
	filesMeta, totalSize, err := calculateFilesMeta()
	if err != nil {
		return err
	}

	// Calculate overall hash
	overallHash, err := calculateOverallHash(filesMeta)
	if err != nil {
		return err
	}

	// Create meta data
	meta := MetaData{
		Hash:      overallHash,
		TotalSize: totalSize,
	}

	// Create files meta data
	filesMetaData := MetaDataForFiles{
		Files: filesMeta,
	}

	// Marshal to JSON
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	filesMetaJSON, err := json.Marshal(filesMetaData)
	if err != nil {
		return err
	}

	// Update cache
	cacheMutex.Lock()
	metaCache = metaJSON
	filesMetaCache = filesMetaJSON
	cacheMutex.Unlock()

	// Write to files
	if err := os.WriteFile(metaFile, metaJSON, 0644); err != nil {
		return err
	}

	if err := os.WriteFile(filesmetaFile, filesMetaJSON, 0644); err != nil {
		return err
	}

	log.Printf("Meta files updated successfully. Total size: %d, Hash: %s", totalSize, overallHash)
	return nil
}

func calculateFilesMeta() ([]MetaForFile, int64, error) {
	var filesMeta []MetaForFile
	var totalSize int64

	err := filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate file hash
		hash, err := calculateFileHash(path)
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(filesDir, path)
		if err != nil {
			return err
		}

		// Use forward slashes for consistency across platforms
		relPath = filepath.ToSlash(relPath)

		fileMeta := MetaForFile{
			Hash: hash,
			Path: relPath,
			Size: info.Size(),
		}

		filesMeta = append(filesMeta, fileMeta)
		totalSize += info.Size()

		return nil
	})

	return filesMeta, totalSize, err
}

func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	for {
		n, err := file.Read(buf)
		if n > 0 {
			hash.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func calculateOverallHash(filesMeta []MetaForFile) (string, error) {
	hash := md5.New()

	for _, fileMeta := range filesMeta {
		// Include file path, hash, and size in the overall hash calculation
		hash.Write([]byte(fileMeta.Path))
		hash.Write([]byte(fileMeta.Hash))

		// Write size as 8 bytes (int64)
		size := fileMeta.Size
		for i := 0; i < 8; i++ {
			hash.Write([]byte{byte(size)})
			size >>= 8
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func metaHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if metaCache == nil {
		http.Error(w, "Meta data not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(metaCache)
}

func filesmetaHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	defer cacheMutex.RUnlock()

	if filesMetaCache == nil {
		http.Error(w, "Files meta data not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(filesMetaCache)
}

func versionHandler(w http.ResponseWriter, r *http.Request) {
	cacheMutex.RLock()
	v := versionCache
	cacheMutex.RUnlock()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"version": v})
}

func adminVersionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}
	var req struct {
		Version string `json:"version"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Version) == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "version is required"})
		return
	}
	v := strings.TrimSpace(req.Version)
	if err := os.WriteFile(versionFile, []byte(v), 0644); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to write version"})
		return
	}
	cacheMutex.Lock()
	versionCache = v
	cacheMutex.Unlock()
	log.Printf("Version updated to %s", v)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"ok": "true", "version": v})
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := map[string]string{"status": "ok"}
	if key := os.Getenv("DEPLOY_KEY"); key != "" {
		resp["deploy_key"] = key
	}
	json.NewEncoder(w).Encode(resp)
}

// ---------- CORS middleware ----------

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// ---------- Admin auth middleware ----------

func adminAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if adminKey == "" {
			http.Error(w, `{"error":"admin not configured"}`, http.StatusServiceUnavailable)
			return
		}

		// Rate limit by IP
		ip := r.RemoteAddr
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			ip = strings.SplitN(fwd, ",", 2)[0]
		}
		if !adminLimiter.allow(strings.TrimSpace(ip)) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "too many requests, try again later"})
			return
		}

		// Basic auth: username "admin", password is the admin key
		user, pass, ok := r.BasicAuth()
		if !ok || user != "admin" || pass != adminKey {
			w.Header().Set("WWW-Authenticate", `Basic realm="admin"`)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
			return
		}

		next(w, r)
	}
}

// ---------- Admin upload handler ----------

func adminUploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	// Limit to 2 GB
	r.Body = http.MaxBytesReader(w, r.Body, 2<<30)

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "file too large or invalid form data"})
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "no file uploaded"})
		return
	}
	defer file.Close()

	filename := filepath.Base(header.Filename)
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".zip" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "only .zip files are supported"})
		return
	}

	// Save to a temp file next to filesDir
	tmpPath := filepath.Join(filepath.Dir(filesDir), "upload-tmp"+ext)
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to save upload"})
		return
	}

	written, err := io.Copy(tmpFile, file)
	tmpFile.Close()
	if err != nil {
		os.Remove(tmpPath)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to save upload"})
		return
	}
	log.Printf("[admin] received %s (%d bytes), extracting...", filename, written)

	// Extract to a temp directory first, then swap
	tmpExtract := filepath.Join(filepath.Dir(filesDir), "files-new")
	os.RemoveAll(tmpExtract)
	os.MkdirAll(tmpExtract, 0755)

	extractErr := extractArchive(tmpPath, tmpExtract)
	os.Remove(tmpPath)

	if extractErr != nil {
		os.RemoveAll(tmpExtract)
		log.Printf("[admin] extraction failed: %v", extractErr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "extraction failed: " + extractErr.Error()})
		return
	}

	// Atomic swap: remove old files, rename new into place
	oldDir := filepath.Dir(filesDir) + "/files-old"
	os.RemoveAll(oldDir)
	os.Rename(filesDir, oldDir)
	if err := os.Rename(tmpExtract, filesDir); err != nil {
		// Rollback if rename fails
		os.Rename(oldDir, filesDir)
		log.Printf("[admin] swap failed: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to replace files"})
		return
	}
	os.RemoveAll(oldDir)

	// Count extracted files
	var fileCount int
	var totalSize int64
	filepath.Walk(filesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		fileCount++
		totalSize += info.Size()
		return nil
	})

	// Regenerate meta immediately
	if err := generateMetaFiles(); err != nil {
		log.Printf("[admin] warning: meta regeneration failed: %v", err)
	}

	log.Printf("[admin] replaced files: %d files, %d bytes total", fileCount, totalSize)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        true,
		"files":     fileCount,
		"totalSize": totalSize,
	})
}

// extractArchive extracts a zip archive using the system unzip command.
func extractArchive(src, dest string) error {
	cmd := exec.Command("unzip", "-o", "-q", src, "-d", dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return os.ErrInvalid
	}
	_ = out
	return nil
}

// ---------- Admin check-space handler ----------

func adminCheckSpaceHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"error": "method not allowed"})
		return
	}

	needStr := r.URL.Query().Get("need")
	var need int64
	for _, c := range needStr {
		if c >= '0' && c <= '9' {
			need = need*10 + int64(c-'0')
		}
	}
	if need <= 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "need parameter required (bytes)"})
		return
	}

	// Check available disk space where files are stored
	available, err := getAvailableSpace(filesDir)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "unable to check disk space"})
		return
	}

	required := int64(float64(need) * 2.5)
	enough := available >= required

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"ok":        enough,
		"available": available,
		"required":  required,
		"enough":    enough,
	})
}

// getAvailableSpace returns available bytes on the filesystem containing dir.
func getAvailableSpace(dir string) (int64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(dir, &stat); err != nil {
		return 0, err
	}
	return int64(stat.Bavail) * int64(stat.Bsize), nil
}
