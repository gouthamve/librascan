-- name: GetAuthors :many
SELECT name FROM authors WHERE isbn = ?;

-- name: GetCategories :many
SELECT name FROM categories WHERE isbn = ?;

-- name: InsertAuthor :exec
INSERT OR IGNORE INTO authors (isbn, name) VALUES (?, ?);

-- name: InsertCategory :exec
INSERT OR IGNORE INTO categories (isbn, name) VALUES (?, ?);

-- name: CountAuthors :one
SELECT COUNT(*) FROM authors WHERE isbn = ?;

-- name: CountCategories :one
SELECT COUNT(*) FROM categories WHERE isbn = ?;