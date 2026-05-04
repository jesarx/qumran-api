package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

func safeFileName(name string, allowedExts []string) (string, error) {
	clean := filepath.Base(filepath.Clean(name))
	if clean == "." || clean == "/" || clean == "" {
		return "", fmt.Errorf("invalid filename")
	}
	ext := strings.ToLower(filepath.Ext(clean))
	for _, allowed := range allowedExts {
		if ext == allowed {
			return clean, nil
		}
	}
	return "", fmt.Errorf("file type not allowed")
}

func contentDisposition(fileName string) string {
	safe := strings.ReplaceAll(fileName, `"`, `\"`)
	return fmt.Sprintf(`attachment; filename="%s"`, safe)
}

func serveFile(app *application, w http.ResponseWriter, r *http.Request, dir string, allowedExts []string) {
	rawName := r.URL.Query().Get("file")
	if rawName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	fileName, err := safeFileName(rawName, allowedExts)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "invalid file parameter")
		return
	}

	filePath := filepath.Join(dir, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			app.notFoundResponse(w, r)
		} else {
			app.serverErrorResponse(w, r, err)
		}
		return
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	w.Header().Set("Content-Disposition", contentDisposition(fileName))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

func (app *application) serveImages(w http.ResponseWriter, r *http.Request) {
	serveFile(app, w, r, "./uploads/covers", []string{".jpg", ".jpeg", ".png", ".gif", ".webp"})
}

func (app *application) servePdfs(w http.ResponseWriter, r *http.Request) {
	serveFile(app, w, r, "./uploads/pdfs", []string{".pdf"})
}

func (app *application) serveEpubs(w http.ResponseWriter, r *http.Request) {
	serveFile(app, w, r, "./uploads/epubs", []string{".epub"})
}

func (app *application) serveTorrents(w http.ResponseWriter, r *http.Request) {
	serveFile(app, w, r, "./uploads/torrents", []string{".torrent"})
}
