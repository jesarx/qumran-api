package main

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"qumran.jesarx.com/internal/data"
	"qumran.jesarx.com/internal/validator"
)

// CREATE BOOK
func (app *application) createBookHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title        string   `json:"title"`
		ShortTitle   string   `json:"short_title"`
		Tags         []string `json:"tags"`
		Year         int32    `json:"year"`
		AuthorID     int64    `json:"author_id"`
		Author2ID    *int64   `json:"author2_id"`
		PublisherID  int64    `json:"publisher_id"`
		ISBN         string   `json:"isbn"`
		Description  string   `json:"description"`
		Pages        int32    `json:"pages"`
		DirDwl       bool     `json:"dir_dwl"`
		ExternalLink string   `json:"external_link"`
	}

	// FILE UPLOAD

	err := app.readJSONFromForm(w, r, "data", &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	result, err := app.processFiles(w, r, "pdf", "image", input.ShortTitle, input.AuthorID, input.PublisherID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	baseFilename := result["filename"]
	cid := result["cid"]

	book := &data.Book{
		Title:        input.Title,
		ShortTitle:   input.ShortTitle,
		Year:         input.Year,
		Tags:         input.Tags,
		AuthorID:     input.AuthorID,
		Author2ID:    input.Author2ID,
		PublisherID:  input.PublisherID,
		Filename:     baseFilename,
		ISBN:         input.ISBN,
		Description:  input.Description,
		Pages:        input.Pages,
		DirDwl:       input.DirDwl,
		ExternalLink: input.ExternalLink,
		Cid:          cid,
	}

	v := validator.New()

	if data.ValidateBook(v, book); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Books.Insert(book)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	completeBook, err := app.models.Books.GetByID(book.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/books/%d", completeBook.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"book": completeBook}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// SHOW BOOK
func (app *application) showBookHandler(w http.ResponseWriter, r *http.Request) {
	slug, err := app.readSlugParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	book, err := app.models.Books.GetBySlug(slug)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encoutered a problem and could not process your request", http.StatusInternalServerError)
	}
}

// UPDATE BOOK
func (app *application) updateBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}
	book, err := app.models.Books.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	var input struct {
		Title        *string  `json:"title"`
		ShortTitle   *string  `json:"short_title"`
		Tags         []string `json:"tags"`
		Year         *int32   `json:"year"`
		AuthorID     *int64   `json:"author_id"`
		Author2ID    *int64   `json:"author2_id"`
		PublisherID  *int64   `json:"publisher_id"`
		ISBN         *string  `json:"isbn"`
		Description  *string  `json:"description"`
		Pages        *int32   `json:"pages"`
		DirDwl       *bool    `json:"dir_dwl"`
		ExternalLink *string  `json:"external_link"`
	}

	// FILE UPLOAD
	err = app.readJSONFromForm(w, r, "data", &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	result, err := app.processFiles(w, r, "pdf", "image",
		func() string {
			if input.ShortTitle != nil {
				return *input.ShortTitle
			}
			return book.ShortTitle
		}(),
		func() int64 {
			if input.AuthorID != nil {
				return *input.AuthorID
			}
			return book.AuthorID
		}(),
		func() int64 {
			if input.PublisherID != nil {
				return *input.PublisherID
			}
			return book.PublisherID
		}())
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Update book fields if they are provided
	if input.Title != nil {
		book.Title = *input.Title
	}
	if input.ShortTitle != nil {
		book.ShortTitle = *input.ShortTitle
	}
	if input.Year != nil {
		book.Year = *input.Year
	}
	if input.Tags != nil {
		book.Tags = input.Tags
	}
	if input.AuthorID != nil {
		book.AuthorID = *input.AuthorID
	}
	if input.Author2ID != nil {
		book.Author2ID = input.Author2ID
	}
	if input.PublisherID != nil {
		book.PublisherID = *input.PublisherID
	}
	if input.ISBN != nil {
		book.ISBN = *input.ISBN
	}
	if input.Description != nil {
		book.Description = *input.Description
	}
	if input.Pages != nil {
		book.Pages = *input.Pages
	}
	if input.DirDwl != nil {
		book.DirDwl = *input.DirDwl
	}
	if input.ExternalLink != nil {
		book.ExternalLink = *input.ExternalLink
	}

	// If a new file was uploaded, update the filename
	if baseFilename, exists := result["filename"]; exists && baseFilename != "" {
		book.Filename = baseFilename
	}

	v := validator.New()
	if data.ValidateBook(v, book); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Books.Update(book)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrEditConflict):
			app.editConflictResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	completeBook, err := app.models.Books.GetByID(book.ID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"book": completeBook}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// DELETE BOOK
func (app *application) deleteBookHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// First, retrieve the book to get its filename and check existence
	book, err := app.models.Books.GetByID(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	// Define base paths for different file types
	basePath := book.Filename
	uploadDirs := map[string]string{
		"pdf":         "./uploads/pdfs",
		"cover":       "./uploads/covers",
		"pdf_torrent": "./uploads/torrents",
	}

	// Files to delete
	filesToDelete := []string{
		filepath.Join(uploadDirs["pdf"], basePath+".pdf"),
		filepath.Join(uploadDirs["cover"], basePath+".jpg"),
		filepath.Join(uploadDirs["pdf_torrent"], basePath+".pdf.torrent"),
	}

	// Delete associated files
	for _, filePath := range filesToDelete {
		if err := os.Remove(filePath); err != nil {
			if !os.IsNotExist(err) {
				app.logger.Error("Error deleting file %s: %v", filePath, err)
			}
		}
	}

	// Delete the book record from the database
	err = app.models.Books.Delete(id)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "book successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listBookHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title    string
		AuthSlug string
		PubSlug  string
		Tags     []string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Title = app.readString(qs, "title", "")
	input.AuthSlug = app.readString(qs, "authslug", "")
	input.PubSlug = app.readString(qs, "pubslug", "")

	input.Tags = app.readCSV(qs, "tags", []string{})

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"id", "title", "year", "tags", "-id", "-title", "-year", "-tags", "created_at", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	books, metadata, err := app.models.Books.GetAll(input.Title, input.AuthSlug, input.PubSlug, input.Tags, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"books": books, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
