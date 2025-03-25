package main

import (
	"errors"
	"fmt"
	"net/http"

	"qumran.jesarx.com/internal/data"
	"qumran.jesarx.com/internal/validator"
)

func (app *application) listPublishersHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string
		data.Filters
	}

	v := validator.New()

	qs := r.URL.Query()

	input.Name = app.readString(qs, "name", "")

	input.Filters.Page = app.readInt(qs, "page", 1, v)
	input.Filters.PageSize = app.readInt(qs, "page_size", 20, v)

	input.Filters.Sort = app.readString(qs, "sort", "name")
	input.Filters.SortSafelist = []string{"id", "name", "-id", "-name", "book_count", "-book_count"}

	if data.ValidateFilters(v, input.Filters); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	publishers, metadata, err := app.models.Publishers.GetAll(input.Name, input.Filters)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"publishers": publishers, "metadata": metadata}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

// SHOW PUBLISHER HANDLER

func (app *application) showPublisherHandler(w http.ResponseWriter, r *http.Request) {
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

	publisher, books, metadata, err := app.models.Publishers.Get(id, input.Filters)
	if err != nil {
		switch {
		case errors.Is(err, data.ErrRecordNotFound):
			app.notFoundResponse(w, r)
		default:
			app.serverErrorResponse(w, r, err)
		}
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"publisher": publisher, "books": books, "metadata": metadata}, nil)
	if err != nil {
		app.logger.Error(err.Error())
		http.Error(w, "The server encountered a problem and could not process your request", http.StatusInternalServerError)
	}
}

func (app *application) deletePublisherHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the publisher ID from the URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Attempt to delete the publisher
	err = app.models.Publishers.Delete(id)
	if err != nil {
		// Check if the error is due to associated books
		if err.Error() == "publisher not found or has associated books" {
			app.errorResponse(w, r, http.StatusConflict,
				"Cannot delete publisher with associated books")
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

func (app *application) createPublisherHandler(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name string `json:"name"`
	}
	err := app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	publisher := &data.Publisher{
		Name: input.Name,
	}

	v := validator.New()

	if data.ValidatePublisher(v, publisher); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	err = app.models.Publishers.Insert(publisher)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	headers := make(http.Header)
	headers.Set("Location", fmt.Sprintf("/v1/publishers/%d", publisher.ID))

	err = app.writeJSON(w, http.StatusCreated, envelope{"publisher": publisher}, headers)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}

func (app *application) updatePublisherHandler(w http.ResponseWriter, r *http.Request) {
	// Extract the publisher ID from the URL
	id, err := app.readIDParam(r)
	if err != nil {
		app.notFoundResponse(w, r)
		return
	}

	// Parse the input JSON
	var input struct {
		Name string `json:"name"`
	}
	err = app.readJSON(w, r, &input)
	if err != nil {
		app.badRequestResponse(w, r, err)
		return
	}

	// Create a publisher object with the new name
	publisher := &data.Publisher{
		ID:   id,
		Name: input.Name,
	}

	// Validate the publisher
	v := validator.New()
	if data.ValidatePublisher(v, publisher); !v.Valid() {
		app.failedValidationResponse(w, r, v.Errors)
		return
	}

	// Update the publisher
	err = app.models.Publishers.Update(publisher)
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	// Return the updated publisher
	err = app.writeJSON(w, http.StatusOK, envelope{"publisher": publisher}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
