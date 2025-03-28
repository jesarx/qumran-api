package main

import (
	"errors"
	"fmt"
	"net/http"

	"qumran.jesarx.com/internal/data"
	"qumran.jesarx.com/internal/validator"
)

func (app *application) listAuthorsHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string
		LastName string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Name = app.readString(qs, "name", "")
	input.LastName = app.readString(qs, "last_name", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "last_name")
	input.Filters.SortSafelist = []string{"name", "-name", "last_name", "-last_name", "id", "-id", "book_count", "-book_count"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	authors, metadata, err := app.models.Authors.GetAll(input.Name, input.LastName, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"authors": authors, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// SHOW AUTHOR HANDLER

func (app *application) showAuthorHandler(w http.ResponseWriter, r *http.Request) {
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	var input struct {
		ID int64
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "id")
	input.Filters.SortSafelist = []string{"id", "title", "tags", "-id", "-title", "-tags"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	author, books, metadata, err := app.models.Authors.Get(id, input.Filters)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"author": author, "books": books, "metadata": metadata}, nil)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}

func (app *application) createAuthorHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name     string `json:"name"`
		LastName string `json:"last_name"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	author := &data.Author{
		Name:     input.Name,
		LastName: input.LastName,
	}

	v := validator.New()

	if data.ValidateAuthor(v, author); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Authors.Insert(author)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/authors/%d", author.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"author": author}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updateAuthorHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the author ID from the URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Parse the input JSON
	var input struct {
		Name     string `json:"name"`
		LastName string `json:"last_name"`
	}
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Create an author object with the new details
	author := &data.Author{
		ID:       id,
		Name:     input.Name,
		LastName: input.LastName,
	}

	// Validate the author
	v := validator.New()
	if data.ValidateAuthor(v, author); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update the author
	err = app.models.Authors.Update(author)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return the updated author
	err = app.writeJSON(w, http.StatusOK, envelope{"author": author}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) deleteAuthorHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the author ID from the URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Attempt to delete the author
	err = app.models.Authors.Delete(id)
	if err != nil {
		// Check if the error is due to associated books
		if err.Error() == "author not found or has associated books" {
			app.errorResponse(w, r, http.StatusConflict,
				"Cannot delete author with associated books")
			return
		}

		// Handle other potential errors
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return a 204 No Content response on successful deletion
	err = app.writeJSON(w, http.StatusNoContent, envelope{}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
