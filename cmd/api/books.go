package main

import (
	"errors"
	"fmt"
	"net/http"

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
		PublisherID  int64    `json:"publisher_id"`
		ISBN         string   `json:"isbn"`
		Description  string   `json:"description"`
		Pages        int32    `json:"pages"`
		ExternalLink string   `json:"external_link"`
	}

	// FILE UPLOAD

	err := app.readJSONFromForm(w, r, "data", &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	result, err := app.processFiles(w, r, "pdf", "image", input.ShortTitle, input.AuthorID)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	baseFilename := result["filename"]

	book := &data.Book{
		Title:        input.Title,
		ShortTitle:   input.ShortTitle,
		Year:         input.Year,
		Tags:         input.Tags,
		AuthorID:     input.AuthorID,
		PublisherID:  input.PublisherID,
		Filename:     baseFilename,
		ISBN:         input.ISBN,
		Description:  input.Description,
		Pages:        input.Pages,
		ExternalLink: input.ExternalLink,
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
		Title      *string  `json:"title"`
		ShortTitle *string  `json:"short_title"`
		Year       *int32   `json:"year"`
		Tags       []string `json:"tags"`
	}

	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	if input.Title != nil {
		book.Title = *input.Title
	}
	if input.Year != nil {
		book.Year = *input.Year
	}
	if input.ShortTitle != nil {
		book.ShortTitle = *input.ShortTitle
	}
	if input.Tags != nil {
		book.Tags = input.Tags
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

	err = app.writeJSON(w, http.StatusOK, envelope{"book": book}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// DELETE BOOK
func (app *application) deleteMovieHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

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

	err = app.writeJSON(w, http.StatusOK, envelope{"message": "movie successfully deleted"}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) listBookHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Title string
		Tags  []string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Title = app.readString(qs, "title", "")
	input.Tags = app.readCSV(qs, "tags", []string{})

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "-created_at")
	input.Filters.SortSafelist = []string{"id", "title", "year", "tags", "-id", "-title", "-year", "-tags", "created_at", "-created_at"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	books, metadata, err := app.models.Books.GetAll(input.Title, input.Tags, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"books": books, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
