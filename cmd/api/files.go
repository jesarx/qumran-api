package main

import (
	"fmt"
	"net/http"
	"os"
)

func (app *application) serveImages(w http.ResponseWriter, r *http.Request) {
	// Extract the file name from the query parameter
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	// Construct the full file path
	filePath := fmt.Sprintf("../../uploads/covers/%s", fileName)

	// Open the file
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

	// Get file info to set the Content-Length header
	fileInfo, err := file.Stat()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", string(fileInfo.Size()))

	// Serve the file
	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

func (app *application) servePdfs(w http.ResponseWriter, r *http.Request) {
	// Extract the file name from the query parameter
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	// Construct the full file path
	filePath := fmt.Sprintf("../../uploads/pdfs/%s", fileName)

	// Open the file
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

	// Get file info to set the Content-Length header
	fileInfo, err := file.Stat()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", string(fileInfo.Size()))

	// Serve the file
	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

func (app *application) serveEpubs(w http.ResponseWriter, r *http.Request) {
	// Extract the file name from the query parameter
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	// Construct the full file path
	filePath := fmt.Sprintf("../../uploads/epubs/%s", fileName)

	// Open the file
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

	// Get file info to set the Content-Length header
	fileInfo, err := file.Stat()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", string(fileInfo.Size()))

	// Serve the file
	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}

func (app *application) serveTorrents(w http.ResponseWriter, r *http.Request) {
	// Extract the file name from the query parameter
	fileName := r.URL.Query().Get("file")
	if fileName == "" {
		app.errorResponse(w, r, http.StatusBadRequest, "file parameter is required")
		return
	}

	// Construct the full file path
	filePath := fmt.Sprintf("../../uploads/torrents/%s", fileName)

	// Open the file
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

	// Get file info to set the Content-Length header
	fileInfo, err := file.Stat()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Set headers for file download
	w.Header().Set("Content-Disposition", "attachment; filename="+fileName)
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", string(fileInfo.Size()))

	// Serve the file
	http.ServeContent(w, r, fileName, fileInfo.ModTime(), file)
}
