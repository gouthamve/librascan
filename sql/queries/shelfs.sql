-- name: GetShelf :one
SELECT id, name, rows_count FROM shelfs WHERE id = ?;

-- name: GetShelfName :one
SELECT name FROM shelfs WHERE id = ?;

-- name: InsertShelf :exec
INSERT INTO shelfs (name, rows_count) VALUES (?, ?);

-- name: GetAllShelfs :many
SELECT id, name, rows_count FROM shelfs;