package main

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

func safeFileName(name string) (string, error) {
	// Strip any directory components — only allow a bare filename
	clean := filepath.Base(filepath.Clean(name))
	if clean == "." || clean == "/" || clean == "" {
		return "", fmt.Errorf("invalid filename")
	}
	return clean, nil
}

func (app *application) serveImages(w http.ResponseWriter, r *http.Request) {
	rawName := r.URL.Query().Get("file")
	if rawName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	fileName, err := safeFileName(rawName)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "invalid file parameter")
		return
	}

	filePath := filepath.Join("./uploads/covers", fileName)

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

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

func (app *application) servePdfs(w http.ResponseWriter, r *http.Request) {
	rawName := r.URL.Query().Get("file")
	if rawName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	fileName, err := safeFileName(rawName)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "invalid file parameter")
		return
	}

	filePath := filepath.Join("./uploads/pdfs", fileName)

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

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

func (app *application) serveEpubs(w http.ResponseWriter, r *http.Request) {
	rawName := r.URL.Query().Get("file")
	if rawName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	fileName, err := safeFileName(rawName)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "invalid file parameter")
		return
	}

	filePath := filepath.Join("./uploads/epubs", fileName)

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

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

func (app *application) serveTorrents(w http.ResponseWriter, r *http.Request) {
	rawName := r.URL.Query().Get("file")
	if rawName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	fileName, err := safeFileName(rawName)
	if err != nil {
		app.errorResponse(w, r, http.StatusBadRequest, "invalid file parameter")
		return
	}

	filePath := filepath.Join("./uploads/torrents", fileName)

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

	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", strconv.FormatInt(fileInfo.Size(), 10))

	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}
