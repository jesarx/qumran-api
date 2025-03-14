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

type Author struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	LastName  string    `json:"last_name"`
	CreatedAt time.Time `json:"-"`
}

func ValidateAuthor(v *validator.Validator, author *Author) {
	v.Check(author.LastName != "", "last_name", "must be provided")
	v.Check(len(author.LastName) <= 200, "last_name", "must not be more than 500 bytes long")
}

type AuthorModel struct {
	DB *sql.DB
}

func (m AuthorModel) Insert(author *Author) error {
	query := `
    INSERT INTO authors (name, last_name)
    VALUES ($1, $2)
    RETURNING id
  `

	args := []any{author.Name, author.LastName}

	return m.DB.QueryRow(query, args...).Scan(&author.ID)
}

func (m AuthorModel) Get(id int64, filters Filters) (*Author, []*Book, Metadata, error) {
	if id < 1 {
		return nil, nil, Metadata{}, ErrRecordNotFound
	}

	query1 := `
    SELECT id, name, last_name
    FROM authors
    WHERE id = $1;
  `

	query2 := fmt.Sprintf(`
    SELECT count(*) OVER(), id, title, short_title, year, tags, version
    FROM books
    WHERE auth_id = $1
    ORDER by %s %s, title ASC
    LIMIT $2 OFFSET $3
  `, filters.sortColumn(), filters.sortDirection())

	var author Author

	err := m.DB.QueryRow(query1, id).Scan(&author.ID, &author.Name, &author.LastName)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows):
			return nil, nil, Metadata{}, ErrRecordNotFound
		default:
			return nil, nil, Metadata{}, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query2, id, filters.limit(), filters.offset())
	if err != nil {
		return nil, nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	books := []*Book{}

	for rows.Next() {
		var book Book

		err := rows.Scan(&totalRecords, &book.ID, &book.Title, &book.ShortTitle, &book.Year, pq.Array(&book.Tags), &book.Version)
		if err != nil {
			return nil, nil, Metadata{}, err
		}

		books = append(books, &book)
	}

	if err = rows.Err(); err != nil {
		return nil, nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return &author, books, metadata, nil
}

func (m AuthorModel) GetAll(name string, last_name string, filters Filters) ([]*Author, Metadata, error) {
	query := fmt.Sprintf(`
        SELECT count(*) OVER(), id, name, last_name
        FROM authors
        WHERE (to_tsvector('simple', name) @@ plainto_tsquery('simple', $1) OR $1 = '')
        AND (to_tsvector('simple', last_name) @@ plainto_tsquery('simple', $2) OR $2 = '')
        ORDER by %s %s, last_name ASC
        LIMIT $3 OFFSET $4
    `, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{name, last_name, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()
	fmt.Println("Query:", rows)

	totalRecords := 0
	authors := []*Author{}

	for rows.Next() {
		var author Author

		err := rows.Scan(&totalRecords, &author.ID, &author.Name, &author.LastName)
		if err != nil {
			return nil, Metadata{}, err
		}

		authors = append(authors, &author)

	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return authors, metadata, nil
}
