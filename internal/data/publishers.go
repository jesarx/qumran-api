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

type Publisher struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Books     int64     `json:"books"`
	CreatedAt time.Time `json:"-"`
}

func ValidatePublisher(v *validator.Validator, publisher *Publisher) {
	v.Check(publisher.Name != "", "name", "must be provided")
	v.Check(len(publisher.Name) <= 200, "name", "must not be more than 500 bytes long")
}

type PublisherModel struct {
	DB *sql.DB
}

func (m PublisherModel) Insert(publisher *Publisher) error {
	query := `
    INSERT INTO publishers (name)
    VALUES ($1)
    RETURNING id
  `

	args := []any{publisher.Name}

	return m.DB.QueryRow(query, args...).Scan(&publisher.ID)
}

func (m PublisherModel) Get(id int64, filters Filters) (*Publisher, []*Book, Metadata, error) {
	if id < 1 {
		return nil, nil, Metadata{}, ErrRecordNotFound
	}

	query1 := `
    SELECT id, name
    FROM publishers
    WHERE id = $1;
  `

	query2 := fmt.Sprintf(`
    SELECT count(*) OVER(), id, title, short_title, year, tags, version
    FROM books
    WHERE pub_id = $1
    ORDER by %s %s, title ASC
    LIMIT $2 OFFSET $3
  `, filters.sortColumn(), filters.sortDirection())

	var publisher Publisher

	err := m.DB.QueryRow(query1, id).Scan(&publisher.ID, &publisher.Name)
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

	return &publisher, books, metadata, nil
}

func (m PublisherModel) GetAll(name string, filters Filters) ([]*Publisher, Metadata, error) {
	query := fmt.Sprintf(`
    SELECT count(*) OVER(), p.id, p.name, p.slug, COUNT(b.id) as book_count
    FROM publishers p
    LEFT JOIN books b ON p.id = b.pub_id
    WHERE (to_tsvector('simple', p.name) @@ plainto_tsquery('simple', $1) OR $1 = '')
    GROUP BY p.id, p.name
    ORDER by %s %s, p.name ASC
    LIMIT $2 OFFSET $3
`, filters.sortColumn(), filters.sortDirection())

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	args := []any{name, filters.limit(), filters.offset()}

	rows, err := m.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, Metadata{}, err
	}
	defer rows.Close()

	totalRecords := 0
	publishers := []*Publisher{}

	for rows.Next() {
		var publisher Publisher

		err := rows.Scan(&totalRecords, &publisher.ID, &publisher.Name, &publisher.Slug, &publisher.Books)
		if err != nil {
			return nil, Metadata{}, err
		}

		publishers = append(publishers, &publisher)

	}

	if err = rows.Err(); err != nil {
		return nil, Metadata{}, err
	}

	metadata := calculateMetadata(totalRecords, filters.Page, filters.PageSize)

	return publishers, metadata, nil
}
