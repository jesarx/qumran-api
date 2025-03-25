package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
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

func (app *application) readSlugParam(r *http.Request) (string, error) {
	params := httprouter.ParamsFromContext(r.Context())
	slug := params.ByName("slug")

	// Validate that the slug isn't empty and contains valid characters
	if slug == "" {
		return "", errors.New("slug parameter cannot be empty")
	}

	// This regex allows lowercase letters, numbers, and hyphens
	validSlug := regexp.MustCompile(`^[a-z0-9-]+$`).MatchString(slug)
	if !validSlug {
		return "", errors.New("invalid slug format")
	}

	return slug, nil
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

func (app *application) processFiles(w http.ResponseWriter, r *http.Request, pdfField string, imageField string, shortTitle string, authorID int64, publisherID int64) (map[string]string, error) {
	// Define trackers for torrent files (can be modified as needed)
	trackers := []string{
		"udp://tracker.opentrackr.org:1337/announce",
		"udp://open.demonii.com:1337/announce",
		"udp://tracker.torrent.eu.org:451/announce",
	}

	var fileData struct {
		FileName     string
		baseFilename string
		ImagePath    string
		PDFPath      string
		EPUBPath     string
		EPUBTorrPath string
		PDFTorrPath  string
	}

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

	publisher, _, _, err := app.models.Publishers.Get(publisherID, data.Filters{
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
	result := map[string]string{
		"filename": baseFileName,
	}

	// Define the target directories
	pdfTargetDir := "../../uploads/pdfs"
	imageTargetDir := "../../uploads/covers"
	epubTargetDir := "../../uploads/epubs"
	torrentTargetDir := "../../uploads/torrents"

	// Create the directories if they don't exist
	for _, dir := range []string{pdfTargetDir, imageTargetDir, epubTargetDir, torrentTargetDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	// PDF Processing (Optional)
	pdfFile, pdfHeader, err := r.FormFile(pdfField)
	if err == nil {
		defer pdfFile.Close()

		// Define PDF file details
		pdfExtension := filepath.Ext(pdfHeader.Filename)
		fileData.FileName = baseFileName + pdfExtension
		fileData.PDFPath = filepath.Join(pdfTargetDir, fileData.FileName)
		fileData.EPUBPath = filepath.Join(epubTargetDir, baseFileName+".epub")
		fileData.PDFTorrPath = filepath.Join(torrentTargetDir, baseFileName+".pdf.torrent")
		fileData.EPUBTorrPath = filepath.Join(torrentTargetDir, baseFileName+".epub.torrent")

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

		// Run exiftool on the PDF file to remove all metadata
		pdfExifCmd := exec.Command("exiftool",
			"-overwrite_original",
			"-all:all=", fileData.PDFPath)
		pdfExifOutput, err := pdfExifCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to run exiftool on PDF: %w, output: %s", err, string(pdfExifOutput))
		}

		// Now add specific metadata to the PDF
		pdfMetadataCmd := exec.Command("exiftool",
			"-overwrite_original",
			"-charset", "exif=UTF8",
			"-Title="+shortTitle,
			"-Author="+author.Name+" "+author.LastName,
			"-Publisher="+publisher.Name,
			fileData.PDFPath)

		pdfMetadataOutput, err := pdfMetadataCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to add metadata to PDF: %w, output: %s", err, string(pdfMetadataOutput))
		}

		// Create EPUB from PDF using ebook-convert
		ebookConvertCmd := exec.Command("ebook-convert",
			fileData.PDFPath,
			fileData.EPUBPath,
			"--no-images",
			"--enable-heuristics",
			"--remove-paragraph-spacing",
			"--base-font-size=12",
			"--asciiize",
			"--title="+shortTitle,
			"--authors="+author.Name+" "+author.LastName,
			"--publisher="+publisher.Name)

		ebookConvertOutput, err := ebookConvertCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to convert PDF to EPUB: %w, output: %s", err, string(ebookConvertOutput))
		}

		// Create torrent file for PDF using transmission-create
		trackersArg := strings.Join(trackers, ",")
		pdfTorrentCmd := exec.Command("transmission-create",
			"-o", fileData.PDFTorrPath,
			"-c", fmt.Sprintf("%s by %s %s", shortTitle, author.Name, author.LastName),
			"-t", trackersArg,
			fileData.PDFPath)

		pdfTorrentOutput, err := pdfTorrentCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to create PDF torrent: %w, output: %s", err, string(pdfTorrentOutput))
		}

		// Create torrent file for EPUB using transmission-create
		epubTorrentCmd := exec.Command("transmission-create",
			"-o", fileData.EPUBTorrPath,
			"-c", fmt.Sprintf("%s by %s %s", shortTitle, author.Name, author.LastName),
			"-t", trackersArg,
			fileData.EPUBPath)

		epubTorrentOutput, err := epubTorrentCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to create EPUB torrent: %w, output: %s", err, string(epubTorrentOutput))
		}

		// Add PDF-related files to result
		result["pdf"] = fileData.PDFPath
		result["epub"] = fileData.EPUBPath
		result["pdf_torrent"] = fileData.PDFTorrPath
		result["epub_torrent"] = fileData.EPUBTorrPath
	}

	// Image Processing (Optional)
	imageFile, _, err := r.FormFile(imageField)
	if err == nil {
		defer imageFile.Close()

		// Define image file details
		imageExtension := ".jpg"
		imageFileName := baseFileName + imageExtension
		fileData.ImagePath = filepath.Join(imageTargetDir, imageFileName)

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

		// Run exiftool on the image file to remove all metadata
		imageExifCmd := exec.Command("exiftool",
			"-overwrite_original",
			"-all:all=", fileData.ImagePath)
		imageExifOutput, err := imageExifCmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("failed to run exiftool on image: %w, output: %s", err, string(imageExifOutput))
		}

		// Add image-related file to result
		result["image"] = fileData.ImagePath

		// Update EPUB cover if PDF was processed earlier
		if _, pdfProcessed := result["pdf"]; pdfProcessed {
			// Recreate EPUB with cover image
			ebookConvertCmd := exec.Command("ebook-convert",
				fileData.PDFPath,
				fileData.EPUBPath,
				"--no-images",
				"--enable-heuristics",
				"--remove-paragraph-spacing",
				"--base-font-size=12",
				"--asciiize",
				"--title="+shortTitle,
				"--authors="+author.Name+" "+author.LastName,
				"--publisher="+publisher.Name,
				"--cover="+fileData.ImagePath)

			ebookConvertOutput, err := ebookConvertCmd.CombinedOutput()
			if err != nil {
				return nil, fmt.Errorf("failed to update EPUB with cover: %w, output: %s", err, string(ebookConvertOutput))
			}

			result["epub"] = fileData.EPUBPath
		}
	}

	return result, nil
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
