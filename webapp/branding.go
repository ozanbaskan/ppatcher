package main

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/gif"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	xdraw "golang.org/x/image/draw"
)

const (
	maxBrandUploadBytes = 10 << 20 // 10MB input guard
	logoTargetBytes     = 250 * 1024
	iconTargetBytes     = 125 * 1024
	logoMaxDimension    = 1200
	iconMaxDimension    = 512
)

func handleUploadBrandAsset(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	kind := strings.ToLower(mux.Vars(r)["kind"])
	var targetBytes, maxDim int
	var updateSQL string
	switch kind {
	case "logo":
		targetBytes = logoTargetBytes
		maxDim = logoMaxDimension
		updateSQL = `UPDATE applications SET logo_path=$1, updated_at=$2 WHERE id=$3 AND user_id=$4`
	case "icon":
		targetBytes = iconTargetBytes
		maxDim = iconMaxDimension
		updateSQL = `UPDATE applications SET icon_path=$1, updated_at=$2 WHERE id=$3 AND user_id=$4`
	default:
		writeJSON(w, 400, apiResponse{Error: "Invalid asset type"})
		return
	}

	if err := r.ParseMultipartForm(maxBrandUploadBytes); err != nil {
		writeJSON(w, 400, apiResponse{Error: "File too large (max 10MB upload)"})
		return
	}
	file, _, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, 400, apiResponse{Error: "No file uploaded"})
		return
	}
	defer file.Close()

	raw, err := io.ReadAll(io.LimitReader(file, maxBrandUploadBytes+1))
	if err != nil {
		writeJSON(w, 500, apiResponse{Error: "Failed to read uploaded file"})
		return
	}
	if len(raw) == 0 {
		writeJSON(w, 400, apiResponse{Error: "Uploaded file is empty"})
		return
	}
	if len(raw) > maxBrandUploadBytes {
		writeJSON(w, 400, apiResponse{Error: "File too large (max 10MB upload)"})
		return
	}

	out, err := compressBrandImage(raw, targetBytes, maxDim)
	if err != nil {
		writeJSON(w, 400, apiResponse{Error: err.Error()})
		return
	}

	brandDir := filepath.Join(appDistDir(app.ID), "branding")
	if err := os.MkdirAll(brandDir, 0755); err != nil {
		writeJSON(w, 500, apiResponse{Error: "Failed to create branding directory"})
		return
	}

	filePath := filepath.Join(brandDir, kind+".jpg")
	if err := os.WriteFile(filePath, out, 0644); err != nil {
		writeJSON(w, 500, apiResponse{Error: "Failed to save image"})
		return
	}

	if _, err := db.Exec(r.Context(), updateSQL, filePath, time.Now(), app.ID, user.ID); err != nil {
		writeJSON(w, 500, apiResponse{Error: "Failed to save image metadata"})
		return
	}

	writeJSON(w, 200, apiResponse{OK: true, Data: map[string]interface{}{
		"url":  fmt.Sprintf("/apps/%d/assets/%s?v=%d", app.ID, kind, time.Now().Unix()),
		"size": len(out),
	}})
}

func handleGetBrandAsset(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	app, err := getAppForUser(r, user.ID)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	kind := strings.ToLower(mux.Vars(r)["kind"])
	var filePath string
	switch kind {
	case "logo":
		filePath = app.LogoPath
	case "icon":
		filePath = app.IconPath
	default:
		http.NotFound(w, r)
		return
	}

	if filePath == "" {
		http.NotFound(w, r)
		return
	}
	if _, err := os.Stat(filePath); err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Cache-Control", "no-cache")
	http.ServeFile(w, r, filePath)
}

func compressBrandImage(raw []byte, maxBytes, maxDimension int) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("invalid image file")
	}

	resized := fitWithin(img, maxDimension)
	best, bestSize := encodeBestJPEG(resized)
	if bestSize <= maxBytes {
		return best, nil
	}

	current := resized
	for i := 0; i < 10; i++ {
		w := current.Bounds().Dx()
		h := current.Bounds().Dy()
		if w <= 64 || h <= 64 {
			break
		}
		current = scaleBy(current, 0.85)
		candidate, candidateSize := encodeBestJPEG(current)
		if candidateSize < bestSize {
			best = candidate
			bestSize = candidateSize
		}
		if candidateSize <= maxBytes {
			return candidate, nil
		}
	}

	if bestSize > maxBytes {
		return nil, fmt.Errorf("could not shrink image under %dKB; try a simpler image", maxBytes/1024)
	}
	return best, nil
}

func encodeBestJPEG(img image.Image) ([]byte, int) {
	qualities := []int{90, 82, 74, 66, 58, 50, 42, 34, 26, 20}
	var best []byte
	bestSize := 0
	for i, q := range qualities {
		var buf bytes.Buffer
		_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: q})
		sz := buf.Len()
		if i == 0 || sz < bestSize {
			best = append([]byte(nil), buf.Bytes()...)
			bestSize = sz
		}
	}
	return best, bestSize
}

func fitWithin(img image.Image, maxDimension int) image.Image {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	maxSide := w
	if h > maxSide {
		maxSide = h
	}
	if maxSide <= maxDimension {
		return img
	}
	scale := float64(maxDimension) / float64(maxSide)
	newW := int(float64(w) * scale)
	newH := int(float64(h) * scale)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	return resizeImage(img, newW, newH)
}

func scaleBy(img image.Image, factor float64) image.Image {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	newW := int(float64(w) * factor)
	newH := int(float64(h) * factor)
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	return resizeImage(img, newW, newH)
}

func resizeImage(src image.Image, w, h int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}
