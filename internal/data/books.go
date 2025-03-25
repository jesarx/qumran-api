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
	ID              int64     `json:"id"`
	CreatedAt       time.Time `json:"-"`
	Year            int32     `json:"year,omitempty"`
	Title           string    `json:"title,omitempty"`
	ShortTitle      string    `json:"short_title,omitempty"`
	Tags            []string  `json:"tags,omitempty"`
	AuthorID        int64     `json:"author_id,omitempty"`
	AuthorName      string    `json:"author_name,omitempty"`
	AuthorLastName  string    `json:"author_last_name,omitempty"`
	AuthorSlug      string    `json:"author_slug,omitempty"`
	Author2ID       *int64    `json:"author2_id,omitempty"`
	Author2Name     *string   `json:"author2_name,omitempty"`
	Author2LastName *string   `json:"author2_last_name,omitempty"`
	Author2Slug     *string   `json:"author2_slug,omitempty"`
	PublisherID     int64     `json:"publisher_id,omitempty"`
	PublisherName   string    `json:"publisher_name,omitempty"`
	PublisherSlug   string    `json:"publisher_slug,omitempty"`
	DirDwl          bool      `json:"dir_dwl,omitempty"`
	Slug            string    `json:"slug,omitempty"`
	Version         int32     `json:"version"`
	Filename        string    `json:"filename,omitempty"`
	ISBN            string    `json:"isbn,omitempty"`
	Description     string    `json:"description,omitempty"`
	Pages           int32     `json:"pages,omitempty"`
	ExternalLink    string    `json:"external_link,omitempty"`
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
    INSERT INTO books (title, short_title, year, tags, auth_id, auth2_id, pub_id, filename, isbn, description, pages, external_link)
    VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    RETURNING id, created_at
  `

	args := []any{book.Title, book.ShortTitle, book.Year, pq.Array(book.Tags), book.AuthorID, book.Author2ID, book.PublisherID, book.Filename, book.ISBN, book.Description, book.Pages, book.ExternalLink}

	return b.DB.QueryRow(query, args...).Scan(&book.ID, &book.CreatedAt)
}

func (b BookModel) GetByID(id int64) (*Book, error) {
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
      b.slug,
      b.filename
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
		&book.ID, &book.CreatedAt, &book.Title, &book.ShortTitle, &book.Year, pq.Array(&book.Tags), &book.AuthorID, &book.AuthorName, &book.AuthorLastName, &book.PublisherID, &book.PublisherName, &book.Version, &book.Slug, &book.Filename,
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

func (b BookModel) GetBySlug(slug string) (*Book, error) {
	if slug == "" {
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
  a1.name AS author_name, 
  a1.last_name AS author_last_name,
  a1.slug AS author_slug,
  b.auth2_id,
  a2.name AS author_name_2,
  a2.last_name AS author_last_name_2,
  a2.slug AS author2_slug,
  b.pub_id, 
  p.name AS publisher_name,
  p.slug AS publisher_slug,
  b.version,
  b.slug,
  b.filename,
  b.description,
  b.pages,
  b.isbn,
  b.external_link,
  b.dir_dwl
FROM 
  books b
JOIN 
  authors a1 ON b.auth_id = a1.id
LEFT JOIN 
  authors a2 ON b.auth2_id = a2.id
JOIN 
  publishers p ON b.pub_id = p.id
WHERE 
  b.slug = $1;
  `

	var book Book
	err := b.DB.QueryRow(query, slug).Scan(
		&book.ID, &book.CreatedAt, &book.Title, &book.ShortTitle, &book.Year, pq.Array(&book.Tags),
		&book.AuthorID, &book.AuthorName, &book.AuthorLastName, &book.AuthorSlug, &book.Author2ID, &book.Author2Name, &book.Author2LastName, &book.Author2Slug, &book.PublisherID,
		&book.PublisherName, &book.PublisherSlug, &book.Version, &book.Slug, &book.Filename, &book.Description, &book.Pages, &book.ISBN, &book.ExternalLink, &book.DirDwl,
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
    SET 
        title = $1, 
        short_title = $2, 
        year = $3, 
        tags = $4, 
        auth_id = $5,
        auth2_id = $6,
        pub_id = $7,
        filename = $8,
        isbn = $9,
        description = $10,
        pages = $11,
        dir_dwl = $12,
        external_link = $13,
        version = version + 1
    WHERE id = $14 AND version = $15
    RETURNING version
  `
	args := []any{
		book.Title,
		book.ShortTitle,
		book.Year,
		pq.Array(book.Tags),
		book.AuthorID,
		book.Author2ID,
		book.PublisherID,
		book.Filename,
		book.ISBN,
		book.Description,
		book.Pages,
		book.DirDwl,
		book.ExternalLink,
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

func (b BookModel) GetAll(title string, authslug string, pubslug string, tags []string, filters Filters) ([]*Book, Metadata, error) {
	query := fmt.Sprintf(`
    SELECT 
        count(*) OVER(),
        b.id, 
        b.created_at, 
        b.title, 
        b.short_title, 
        b.auth_id, 
        b.auth2_id,
        b.pub_id, 
        b.year, 
        b.tags, 
        b.slug, 
        b.filename,
        b.version,
        a.name AS author_name,
        a.last_name AS author_last_name,
        a.slug AS author_slug,
        a2.name AS author_name_2,
        a2.last_name AS author_last_name_2,
        a2.slug AS author2_slug,
        p.name AS publisher_name,
        p.slug AS publisher_slug,
        b.dir_dwl
    FROM 
        books b
    JOIN 
        authors a ON b.auth_id = a.id
    LEFT JOIN
        authors a2 ON b.auth2_id = a2.id
    JOIN 
        publishers p ON b.pub_id = p.id
    WHERE 
        (to_tsvector('spanish', unaccent(b.title)) @@ plainto_tsquery('spanish', unaccent($1)) OR $1 = '') 
        AND (b.tags @> $2 OR $2 = '{}')
        AND ($5 = '' OR a.slug = $5 OR a2.slug = $5)
        AND ($6 = '' OR p.slug = $6)
    ORDER BY 
        %s %s, b.title ASC
    LIMIT $3 OFFSET $4
`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{title, pq.Array(tags), filters.limit(), filters.offset(), authslug, pubslug}

	rows, err := b.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}

	defer rows.Close()

	totalRecords := 0
	books := []*Book{}

	for rows.Next() {
		var book Book

		err := rows.Scan(
			&totalRecords,
			&book.ID,
			&book.CreatedAt,
			&book.Title,
			&book.ShortTitle,
			&book.AuthorID,
			&book.Author2ID,
			&book.PublisherID,
			&book.Year,
			pq.Array(&book.Tags),
			&book.Slug,
			&book.Filename,
			&book.Version,
			&book.AuthorName,
			&book.AuthorLastName,
			&book.AuthorSlug,
			&book.Author2Name,
			&book.Author2LastName,
			&book.Author2Slug,
			&book.PublisherName,
			&book.PublisherSlug,
			&book.DirDwl)
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
