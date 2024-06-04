package data

import (
	"database/sql"
	"errors"
	"time"

	"github.com/lib/pq"
	"qumran.jesarx.com/internal/validator"
)

type Book struct {
	ID         int64     `json:"id"`
	CreatedAt  time.Time `json:"created_at"`
	Year       int32     `json:"year"`
	Title      string    `json:"title"`
	ShortTitle string    `json:"short_title"`
	Tags       []string  `json:"tags"`
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

	v.Check(validator.Unique(book.Tags), "tags", "must not contain diplicate values")
}

type BookModel struct {
	DB *sql.DB
}

func (b BookModel) Insert(book *Book) error {
	query := `
    INSERT INTO books (title, short_title, year, tags)
    VALUES ($1, $2, $3, $4)
    RETURNING id, created_at
  `

	args := []any{book.Title, book.ShortTitle, book.Year, pq.Array(book.Tags)}

	return b.DB.QueryRow(query, args...).Scan(&book.ID, &book.CreatedAt)
}

func (b BookModel) Get(id int64) (*Book, error) {
	if id < 1 {
		return nil, ErrRecordNotFound
	}

	query := `
    SELECT id, created_at, title, short_title, year, tags
    FROM books
    WHERE id = $1
  `

	var book Book

	err := b.DB.QueryRow(query, id).Scan(
		&book.ID, &book.CreatedAt, &book.Title, &book.ShortTitle, &book.Year, pq.Array(&book.Tags),
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
    SET title = $1, short_title = $2, year = $3, tags = $4
    WHERE id = $5
    RETURNING id
  `
	args := []any{
		book.Title,
		book.ShortTitle,
		book.Year,
		pq.Array(book.Tags),
		book.ID,
	}

	return b.DB.QueryRow(query, args...).Scan(&book.ID)
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
