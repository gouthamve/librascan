// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.29.0
// source: people_borrowing.sql

package db

import (
	"context"
)

const getActiveBorrowings = `-- name: GetActiveBorrowings :many
SELECT b.id, b.isbn, b.person_id, b.borrowed_at, p.name as person_name
FROM borrowing b
JOIN people p ON b.person_id = p.id
WHERE b.returned_at IS NULL
`

type GetActiveBorrowingsRow struct {
	ID         int64  `json:"id"`
	Isbn       int64  `json:"isbn"`
	PersonID   int64  `json:"person_id"`
	BorrowedAt string `json:"borrowed_at"`
	PersonName string `json:"person_name"`
}

func (q *Queries) GetActiveBorrowings(ctx context.Context) ([]GetActiveBorrowingsRow, error) {
	rows, err := q.db.QueryContext(ctx, getActiveBorrowings)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []GetActiveBorrowingsRow{}
	for rows.Next() {
		var i GetActiveBorrowingsRow
		if err := rows.Scan(
			&i.ID,
			&i.Isbn,
			&i.PersonID,
			&i.BorrowedAt,
			&i.PersonName,
		); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getAllPeople = `-- name: GetAllPeople :many
SELECT id, name FROM people
`

func (q *Queries) GetAllPeople(ctx context.Context) ([]Person, error) {
	rows, err := q.db.QueryContext(ctx, getAllPeople)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Person{}
	for rows.Next() {
		var i Person
		if err := rows.Scan(&i.ID, &i.Name); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

const getPerson = `-- name: GetPerson :one
SELECT id FROM people WHERE name = ?
`

func (q *Queries) GetPerson(ctx context.Context, name string) (int64, error) {
	row := q.db.QueryRowContext(ctx, getPerson, name)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const insertBorrowing = `-- name: InsertBorrowing :exec
INSERT INTO borrowing (isbn, person_id, borrowed_at) VALUES (?, ?, datetime('now'))
`

type InsertBorrowingParams struct {
	Isbn     int64 `json:"isbn"`
	PersonID int64 `json:"person_id"`
}

func (q *Queries) InsertBorrowing(ctx context.Context, arg InsertBorrowingParams) error {
	_, err := q.db.ExecContext(ctx, insertBorrowing, arg.Isbn, arg.PersonID)
	return err
}

const insertPerson = `-- name: InsertPerson :one
INSERT INTO people (name) VALUES (?) RETURNING id
`

func (q *Queries) InsertPerson(ctx context.Context, name string) (int64, error) {
	row := q.db.QueryRowContext(ctx, insertPerson, name)
	var id int64
	err := row.Scan(&id)
	return id, err
}

const returnBook = `-- name: ReturnBook :exec
UPDATE borrowing 
SET returned_at = datetime('now') 
WHERE isbn = ? AND person_id = ? AND returned_at IS NULL
`

type ReturnBookParams struct {
	Isbn     int64 `json:"isbn"`
	PersonID int64 `json:"person_id"`
}

func (q *Queries) ReturnBook(ctx context.Context, arg ReturnBookParams) error {
	_, err := q.db.ExecContext(ctx, returnBook, arg.Isbn, arg.PersonID)
	return err
}
