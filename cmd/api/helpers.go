package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/julienschmidt/httprouter"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
	"qumran.jesarx.com/internal/data"
	"qumran.jesarx.com/internal/validator"
)

func (app *application) readIDParam(r *http.Request) (int64, error) {
	params := httprouter.ParamsFromContext(r.Context())

	id, err := strconv.ParseInt(params.ByName("id"), 10, 64)
	if err != nil || id < 1 {
		return 0, errors.New("invalid ID parameter")
	}
	return id, nil
}

type envelope map[string]any

func (app *application) writeJSON(w http.ResponseWriter, status int, data envelope, headers http.Header) error {
	js, err := json.Marshal(data)
	if err != nil {
		return err
	}

	js = append(js, '\n')

	for key, value := range headers {
		w.Header()[key] = value
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(js)

	return nil
}

func (app *application) readJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	maxBytes := 1_048_576
	r.Body = http.MaxBytesReader(w, r.Body, int64(maxBytes))
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	err := dec.Decode(dst)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		var invalidUnmarshalError *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &syntaxError):
			return fmt.Errorf("body contains badly-formed JSON (at characcter %d)", syntaxError.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.New("body contains badly-formed JSON")

		case errors.As(err, &unmarshalTypeError):
			if unmarshalTypeError.Field != "" {
				return fmt.Errorf("body contains incorrect JSON type for field %q", unmarshalTypeError.Field)
			}
			return fmt.Errorf("body contains incorrect JSON type (at character %d)", unmarshalTypeError.Offset)

		case errors.Is(err, io.EOF):
			return errors.New("body must not be empty")

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			return fmt.Errorf("body contains unknown key %s", fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("body must not be larger than %d bytes", maxBytesError.Limit)

		case errors.As(err, &invalidUnmarshalError):
			panic(err)

		default:
			return err
		}
	}

	err = dec.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errors.New("body must only contain a single JSON value")
	}

	return nil
}

func (app *application) readJSONFromForm(w http.ResponseWriter, r *http.Request, formField string, dst any) error {
	// Retrieve JSON string from form value
	jsonData := r.FormValue(formField)
	if jsonData == "" {
		return errors.New("form field is empty or missing")
	}

	// Convert JSON string to io.Reader
	jsonReader := strings.NewReader(jsonData)
	r.Body = io.NopCloser(jsonReader)

	// Reuse existing readJSON function
	return app.readJSON(w, r, dst)
}

func (app *application) CleanString(str string) string {
	// Normalize the string to decompose special characters into base characters + diacritics
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, str)

	// Remove any non-alphanumeric characters except spaces
	re := regexp.MustCompile("[^a-zA-Z0-9 ]")
	result := re.ReplaceAllString(normalized, "")

	// Replace spaces with underscores
	return strings.ReplaceAll(result, " ", "_")
}

func (app *application) processFiles(w http.ResponseWriter, r *http.Request, pdfField string, imageField string, shortTitle string, authorID int64) (map[string]string, error) {
	var fileData struct {
		FileName     string
		baseFilename string
		PDFPath      string
		ImagePath    string
	}

	// Retrieve the PDF file
	pdfFile, pdfHeader, err := r.FormFile(pdfField)
	if err != nil {
		app.failedValidationResponse(w, r, map[string]string{"pdf": "PDF file is required"})
		return nil, err
	}
	defer pdfFile.Close()

	// Retrieve the image file
	imageFile, imageHeader, err := r.FormFile(imageField)
	if err != nil {
		app.failedValidationResponse(w, r, map[string]string{"image": "Image file is required"})
		return nil, err
	}
	defer imageFile.Close()

	// Validate the author ID
	if authorID < 1 {
		return nil, errors.New("invalid author ID")
	}

	// Fetch author details
	author, _, _, err := app.models.Authors.Get(authorID, data.Filters{
		Page:         1,
		PageSize:     1,
		Sort:         "id",
		SortSafelist: []string{"id"},
	})
	if err != nil {
		return nil, err
	}

	// Generate a unique filename (without extension)
	baseFileName := fmt.Sprintf("%s_%s-%s", app.CleanString(author.LastName), app.CleanString(author.Name), app.CleanString(shortTitle))

	// Define PDF file details
	pdfExtension := filepath.Ext(pdfHeader.Filename)
	fileData.FileName = baseFileName + pdfExtension

	// Define the target directory
	pdfTargetDir := "../../uploads/covers"

	// Create the directory (and any necessary parent directories) if it doesn't exist
	if err := os.MkdirAll(pdfTargetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	fileData.PDFPath = fmt.Sprintf(pdfTargetDir, fileData.FileName)

	// Define image file details
	imageExtension := ".jpg"
	imageFileName := baseFileName + imageExtension

	// Define the target directory
	imageTargetDir := "../../uploads/covers"

	// Create the directory (and any necessary parent directories) if it doesn't exist
	if err := os.MkdirAll(imageTargetDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	fileData.ImagePath = fmt.Sprintf(imageTargetDir, imageFileName)

	// Save the PDF file
	pdfDst, err := os.Create(fileData.PDFPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF file: %w", err)
	}
	defer pdfDst.Close()

	_, err = io.Copy(pdfDst, pdfFile)
	if err != nil {
		return nil, fmt.Errorf("failed to save PDF file: %w", err)
	}

	// Save the image file
	imageDst, err := os.Create(fileData.ImagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create image file: %w", err)
	}
	defer imageDst.Close()

	_, err = io.Copy(imageDst, imageFile)
	if err != nil {
		return nil, fmt.Errorf("failed to save image file: %w", err)
	}

	// Return success
	return map[string]string{
		"pdf":      fileData.PDFPath,
		"image":    fileData.ImagePath,
		"filename": baseFileName,
	}, nil
}

func (app *application) readString(qs url.Values, key string, defaultValue string) string {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	return s
}

func (app *application) readCSV(qs url.Values, key string, defaultValue []string) []string {
	csv := qs.Get(key)

	if csv == "" {
		return defaultValue
	}

	return strings.Split(csv, ",")
}

func (app *application) readInt(qs url.Values, key string, defaultValue int, v *validator.Validator) int {
	s := qs.Get(key)

	if s == "" {
		return defaultValue
	}

	i, err := strconv.Atoi(s)
	if err != nil {
		v.AddError(key, "must be an integer value")
		return defaultValue
	}

	return i
}

func (app *application) backgound(fn func()) {
	app.wg.Add(1)
	go func() {
		defer app.wg.Done()

		defer func() {
			if err := recover(); err != nil {
				app.logger.Error(fmt.Sprintf("%v", err))
			}
		}()

		fn()
	}()
}
