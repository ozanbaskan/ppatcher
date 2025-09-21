package main

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	goRunTime "runtime"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx  context.Context
	meta MetaData
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	InitConfig()

	runtime.EventsOn(a.ctx, "ready", func(optionalData ...interface{}) {
		a.ManualUpdate()
	})
}

func (a *App) Config() (buildConfig Config) {
	return *BuildConfig
}

func (a *App) tryUpdating() (err error) {

	a.UpdateDownloadStatus("checking")
	ShouldUpdate, err := a.ShouldUpdate()

	if err != nil {
		a.UpdateDownloadStatus("error")
		fmt.Println("Error checking for updates:", err)
		return err
	}
	if ShouldUpdate {
		a.UpdateDownloadStatus("downloading")
		fmt.Println("We should update the files")
		return a.Update()
	}

	a.UpdateDownloadStatus("alreadyReady")

	return nil
}

// Greet returns a greeting for the given name
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, It's show time!", name)
}

type MetaData struct {
	Hash      string `json:"hash"`
	TotalSize int64  `json:"totalSize"`
}

type MetaDataForFiles struct {
	Files []MetaForFile `json:"files"`
}

type MetaForFile struct {
	Hash string
	Path string
	Size int64
}

var (
	bufferPool = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024) // 32KB buffers
		},
	}
)

func calculateFilesMeta() ([]MetaForFile, int64, error) {
	var filesMeta []MetaForFile
	var totalSize int64

	metaDataForFiles, _ := FetchFilesMeta()
	if metaDataForFiles != nil {
		for _, fileMeta := range metaDataForFiles.Files {
			path := fileMeta.Path

			hash, size, err := calculateFileHash(path)
			if err != nil {
				continue
			}

			relPath, err := filepath.Rel("./", path)
			if err != nil {
				continue
			}

			relPath = filepath.ToSlash(relPath)

			fileMeta := MetaForFile{
				Hash: hash,
				Path: relPath,
				Size: size,
			}

			filesMeta = append(filesMeta, fileMeta)
			totalSize += size
		}

		return filesMeta, totalSize, nil
	}

	err := filepath.Walk("./", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Calculate file hash
		hash, _, err := calculateFileHash(path)
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel("./", path)
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

func generateMetaFile() error {
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

	// Marshal to JSON
	metaJSON, err := json.Marshal(meta)
	if err != nil {
		return err
	}

	// Write to files
	if err := os.WriteFile(".downloadmeta", metaJSON, 0644); err != nil {
		return err
	}

	log.Printf("Local meta file updated successfully. Total size: %d, Hash: %s", totalSize, overallHash)
	return nil
}

func calculateFileHash(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	var totalSize int64 = 0
	hash := md5.New()
	buf := bufferPool.Get().([]byte)
	defer bufferPool.Put(buf)

	for {
		n, err := file.Read(buf)
		totalSize += int64(n)
		if n > 0 {
			hash.Write(buf[:n])
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", 0, err
		}
	}

	return hex.EncodeToString(hash.Sum(nil)), totalSize, nil
}
func (a *App) UpdateDownloadStatus(status string) {
	fmt.Println("Download status:", status)
	runtime.EventsEmit(a.ctx, "downloadStatus", status)
}

func (a *App) UpdateDownloadProgress(progress float64) {
	fmt.Println("Download progress:", progress)
	runtime.EventsEmit(a.ctx, "downloadProgress", progress)
}

func (a *App) UpdateCurrentFileData(path string, size int64) {
	runtime.EventsEmit(a.ctx, "currentFileData", map[string]interface{}{
		"path": path,
		"size": size,
	})
}

func (a *App) ShouldUpdate() (should bool, err error) {
	backend := BuildConfig.Backend
	fmt.Println("Checking for updates from backend:", backend)
	resp, err := http.Get(backend + "/meta")

	if err != nil {
		fmt.Println("Error checking for updates:", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error checking for updates: status code", resp.StatusCode)
		return false, fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return false, err
	}

	json.Unmarshal(body, &a.meta)

	fmt.Println("File hash:", a.meta.Hash)
	fmt.Println("File total size:", a.meta.TotalSize)

	data, err := os.ReadFile(".downloadmeta")
	if err != nil {
		fmt.Println("No local meta file found, need to download the files")
		return true, nil
	}

	var localMeta MetaData
	json.Unmarshal(data, &localMeta)

	if localMeta.Hash != a.meta.Hash || localMeta.TotalSize != a.meta.TotalSize {
		fmt.Println(localMeta.Hash, a.meta.Hash)
		fmt.Println("Local meta does not match remote meta, need to download the files")
		return true, nil
	}

	fmt.Println("Local meta matches remote meta, no need to download the files")
	return false, nil
}

func (a *App) ManualUpdate() (err error) {
	err = generateMetaFile()
	if err != nil {
		fmt.Println("Error generating local meta file", err)
		return err
	}

	return a.tryUpdating()
}

func FetchFilesMeta() (filesMeta *MetaDataForFiles, err error) {
	backend := BuildConfig.Backend
	resp, err := http.Get(backend + "/filesmeta")
	if err != nil {
		fmt.Println("Error fetching files meta:", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error fetching files meta: status code", resp.StatusCode)
		return nil, fmt.Errorf("status code %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	json.Unmarshal(body, &filesMeta)

	return filesMeta, nil
}

func (a *App) Update() (err error) {
	backend := BuildConfig.Backend
	fmt.Println("Starting update from backend:", backend)
	filesMeta, err := FetchFilesMeta()
	if err != nil {
		return err
	}

	var totalDownloaded int64 = 0
	var totalSize int64 = a.meta.TotalSize
	var fileCount int64 = int64(len(filesMeta.Files))

	var lastFilePath string = ""
	var lastFileSize int64 = 0

	go func() {
		for {
			remainingFileCount := atomic.LoadInt64(&fileCount)
			if remainingFileCount <= 0 {
				break
			}
			time.Sleep(100 * time.Millisecond)
			a.UpdateDownloadProgress(float64(atomic.LoadInt64(&totalDownloaded)) / float64(totalSize))
			a.UpdateCurrentFileData(lastFilePath, lastFileSize)
		}
	}()

	maxConcurrentDownloads := 10
	semaphore := make(chan struct{}, maxConcurrentDownloads)
	var wg sync.WaitGroup

	for _, file := range filesMeta.Files {
		wg.Add(1)
		file := file

		go func() {
			defer atomic.AddInt64(&totalDownloaded, file.Size)
			defer atomic.AddInt64(&fileCount, -1)

			defer wg.Done()
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			lastFilePath = file.Path
			lastFileSize = file.Size
			hash, _, err := calculateFileHash(file.Path)

			if err != nil {
				fmt.Println("Error calculating the hash for the file", file.Path)
				hash = ""
			}

			if hash == file.Hash {
				fmt.Println("File is up to date, skipping", file.Path)
				return
			}

			err = a.downloadFile(backend, file.Path)
			if err != nil {
				fmt.Println("Error downloading file:", file.Path, err)
			}
		}()
	}

	// Save the new meta file
	respMeta, err := http.Get(backend + "/meta")
	if err != nil {
		fmt.Println("Error fetching meta:", err)
		return err
	}
	defer respMeta.Body.Close()

	if respMeta.StatusCode != http.StatusOK {
		fmt.Println("Error fetching meta: status code", respMeta.StatusCode)
		return fmt.Errorf("status code %d", respMeta.StatusCode)
	}

	metaBody, err := io.ReadAll(respMeta.Body)
	if err != nil {
		fmt.Println("Error reading meta response body:", err)
		return err
	}

	err = os.WriteFile(".downloadmeta", metaBody, 0644)
	if err != nil {
		fmt.Println("Error writing local meta file:", err)
		return err
	}

	wg.Wait()

	a.UpdateDownloadStatus("ready")
	return nil
}

func (a *App) StartExecutable() {
	executablePath := strings.TrimSpace(BuildConfig.Executable)

	if executablePath == "" {
		fmt.Printf("executable path is empty")
		return
	}

	if !filepath.IsAbs(executablePath) {
		absPath, err := filepath.Abs(executablePath)
		if err != nil {
			fmt.Printf("failed to get absolute path: %v", err)
			return
		}
		executablePath = absPath
	}

	// Check if executable exists
	if _, err := os.Stat(executablePath); os.IsNotExist(err) {
		fmt.Printf("executable not found: %s", executablePath)
		return
	}

	fmt.Printf("Starting executable %s\n", executablePath)

	// Start the executable using the absolute path
	cmd := exec.Command(executablePath)
	if err := cmd.Start(); err != nil {
		fmt.Printf("failed to start executable: %v", err)
		return
	}

	// Detach from the process
	go func() {
		err := cmd.Wait()
		if err != nil {
			log.Printf("Executable finished with error: %v", err)
		}
		log.Printf("Executable finished successfully")
	}()
}

func (a *App) downloadFile(backend string, path string) error {
	fmt.Println("Downloading file:", path)

	resp, err := http.Get(backend + "/files/" + path)
	if err != nil {
		fmt.Println("Error downloading file:", path, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Println("Error downloading file: status code", resp.StatusCode)
		return fmt.Errorf("status code %d", resp.StatusCode)
	}

	// Create the directories if they don't exist
	if strings.Contains(path, "/") {
		err = os.MkdirAll(getDir(path), os.ModePerm)
		if err != nil {
			fmt.Println("Error creating directories for file:", path, err)
			return err
		}
	}

	out, err := os.Create(path)
	if err != nil {
		fmt.Println("Error creating file:", path, err)
		return err
	}
	defer out.Close()

	if goRunTime.GOOS != "windows" {
		info, err := out.Stat()
		if err == nil {
			err = out.Chmod(info.Mode() | 0111)
			if err != nil {
				fmt.Printf("Error giving exec permission to file %s", path)
			}
		} else {
			fmt.Printf("Error reading permission of file %s", path)
		}
	}

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		fmt.Println("Error writing file:", path, err)
		return err
	}

	return nil
}

func (a *App) BackendLog(s string) {
	fmt.Println("frontend log:", s)
}

// getDir returns the directory part of a file path
func getDir(path string) string {
	dir := ""
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			dir = path[:i]
			break
		}
	}
	return dir
}
