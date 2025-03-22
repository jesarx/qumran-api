package data

import (
	"context"
	"database/sql"
	"time"
)

type Tag struct {
	Name  string `json:"name"`
	Books int64  `json:"books"`
}

type TagModel struct {
	DB *sql.DB
}

func (m TagModel) GetAll() ([]*Tag, error) {
	query := `
        SELECT 
            t.tag as name,
            COUNT(b.id) as book_count
        FROM 
            (SELECT DISTINCT UNNEST(tags) AS tag FROM books) AS t
        LEFT JOIN 
            books b ON t.tag = ANY(b.tags)
        GROUP BY 
            t.tag
        ORDER BY 
            t.tag ASC
    `

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []*Tag{}
	for rows.Next() {
		var tag Tag
		err := rows.Scan(&tag.Name, &tag.Books)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}
