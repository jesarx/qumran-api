package data

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
	"qumran.jesarx.com/internal/validator"
)

type Book struct {
	ID             int64     `json:"id"`
	CreatedAt      time.Time `json:"-"`
	Year           int32     `json:"year,omitempty"`
	Title          string    `json:"title,omitempty"`
	ShortTitle     string    `json:"short_title,omitempty"`
	Tags           []string  `json:"tags,omitempty"`
	AuthorID       int64     `json:"author_id,omitempty"`
	AuthorName     string    `json:"author_name,omitempty"`
	AuthorLastName string    `json:"author_last_name,omitempty"`
	PublisherID    int64     `json:"publisher_id,omitempty"`
	PublisherName  string    `json:"publisher_name,omitempty"`
	Slug           string    `json:"slug,omitempty"`
	Version        int32     `json:"version"`
	Filename       string    `json:"filename,omitempty"`
	ISBN           string    `json:"isbn,omitempty"`
	Description    string    `json:"description,omitempty"`
	Pages          int32     `json:"pages,omitempty"`
	ExternalLink   string    `json:"external_link,omitempty"`
}

func ValidateBook(v *validator.Validator, book *Book) {
	v.Check(book.Title != "", "title", "must be provided")
	v.Check(len(book.Title) <= 500, "title", "must not be more than 500 bytes long")

	v.Check(book.Year != 0, "year", "must be provided")
	v.Check(book.Year >= 1000, "year", "must be greater than 1000")
	v.Check(book.Year <= int32(time.Now().Year()), "year", "must not be in the future")

	v.Check(book.Tags != nil, "tags", "must be provided")
	v.Check(len(book.Tags) >= 1, "tags", "must contain at least 1 tags")
	v.Check(len(book.Tags) <= 3, "tags", "must not contain more than 3 tags")

	v.Check(book.AuthorID >= 1, "author_id", "must be greater than 1")
	v.Check(book.PublisherID >= 1, "publisher_id", "must be greater than 1")

	v.Check(validator.Unique(book.Tags), "tags", "must not contain diplicate values")
}

type BookModel struct {
	DB *sql.DB
}

func (b BookModel) Insert(book *Book) error {
	query := `
    INSERT INTO books (title, short_title, year, tags, auth_id, pub_id, filename, isbn, description, pages, external_link)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
    RETURNING id, created_at
  `

	args := []any{book.Title, book.ShortTitle, book.Year, pq.Array(book.Tags), book.AuthorID, book.PublisherID, book.Filename, book.ISBN, book.Description, book.Pages, book.ExternalLink}

	return b.DB.QueryRow(query, args...).Scan(&book.ID, &book.CreatedAt)
}

func (b BookModel) Get(id int64) (*Book, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
    SELECT 
      b.id, 
      b.created_at, 
      b.title, 
      b.short_title, 
      b.year, 
      b.tags, 
      b.auth_id, 
      a.name AS author_name, 
      a.last_name AS author_last_name,
      b.pub_id, 
      p.name AS publisher_name,
      b.version,
      b.slug
    FROM 
      books b
    JOIN 
      authors a ON b.auth_id = a.id
    JOIN 
      publishers p ON b.pub_id = p.id
    WHERE 
      b.id = $1;
  `

	var book Book

	err := b.DB.QueryRow(query, id).Scan(
		&book.ID, &book.CreatedAt, &book.Title, &book.ShortTitle, &book.Year, pq.Array(&book.Tags), &book.AuthorID, &book.AuthorName, &book.AuthorLastName, &book.PublisherID, &book.PublisherName, &book.Version, &book.Slug,
	)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, ErrRecordNotFound
		default:
			return nil, err
		}
	}

	return &book, nil
}

func (b BookModel) Update(book *Book) error {
	query := `
    UPDATE books
    SET title = $1, short_title = $2, year = $3, tags = $4, version = version + 1
    WHERE id = $5 AND version = $6
    RETURNING version
  `
	args := []any{
		book.Title,
		book.ShortTitle,
		book.Year,
		pq.Array(book.Tags),
		book.ID,
		book.Version,
	}

	err := b.DB.QueryRow(query, args...).Scan(&book.Version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return ErrEditConflict
		default:
			return err
		}
	}

	return nil
}

func (b BookModel) Delete(id int64) error {
	if id < 1 {
		return ErrRecordNotFound
	}

	query := `
    DELETE FROM books
    WHERE id = $1
  `
	result, err := b.DB.Exec(query, id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

func (b BookModel) GetAll(title string, tags []string, filters Filters) ([]*Book, Metadata, error) {
	query := fmt.Sprintf(`
    SELECT count(*) OVER(), id, created_at, title, short_title, year, tags, slug, version
    FROM books
    WHERE (to_tsvector('spanish', title) @@ plainto_tsquery('spanish', $1) OR $1 = '') 
    AND (tags @> $2 OR $2 = '{}')
    ORDER by %s %s, title ASC
    LIMIT $3 OFFSET $4
    `, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{title, pq.Array(tags), filters.limit(), filters.offset()}

	rows, err := b.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	books := []*Book{}

	for rows.Next() {
		var book Book

		err := rows.Scan(&totalRecords, &book.ID, &book.CreatedAt, &book.Title, &book.ShortTitle, &book.Year, pq.Array(&book.Tags), &book.Slug, &book.Version)
		if err != nil {
			return nil, Metadata{}, err
		}

		books = append(books, &book)

	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return books, metadata, nil
}
