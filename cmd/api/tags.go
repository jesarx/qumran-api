package main

import (
	"net/http"
)

func (app *application) listTagsHandler(w http.ResponseWriter, r *http.Request) {
	tags, err := app.models.Tags.GetAll()
	if err != nil {
		app.serverErrorResponse(w, r, err)
		return
	}

	err = app.writeJSON(w, http.StatusOK, envelope{"tags": tags}, nil)
	if err != nil {
		app.serverErrorResponse(w, r, err)
	}
}
