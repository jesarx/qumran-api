package data

import (
	"time"

	"qumran.jesarx.com/internal/validator"
)

type Book struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Year      int32     `json:"year"`
	Title     string    `json:"title"`
	Tags      []string  `json:"tags"`
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
