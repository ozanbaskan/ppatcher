package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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

const (
	filesDir      = "./files"
	metaFile      = "meta.json"
	filesmetaFile = "filesmeta.json"
	port          = ":3000"
)

var (
	metaCache      []byte
	filesMetaCache []byte
	cacheMutex     sync.RWMutex
	bufferPool     = sync.Pool{
		New: func() interface{} {
			return make([]byte, 32*1024) // 32KB buffers
		},
	}
)

func main() {
	// Create files directory if it doesn't exist
	if err := os.MkdirAll(filesDir, 0755); err != nil {
		log.Fatalf("Failed to create files directory: %v", err)
	}

	// Generate initial meta files
	if err := generateMetaFiles(); err != nil {
		log.Fatalf("Failed to generate initial meta files: %v", err)
	}

	// Start file watcher
	go watchFiles()

	// Set up HTTP handlers
	http.HandleFunc("/meta", metaHandler)
	http.HandleFunc("/filesmeta", filesmetaHandler)
	http.Handle("/files/", http.StripPrefix("/files/", http.FileServer(http.Dir(filesDir))))

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(port, nil))
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
