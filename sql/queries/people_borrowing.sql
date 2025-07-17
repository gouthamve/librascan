-- name: GetPerson :one
SELECT id FROM people WHERE name = ?;

-- name: InsertPerson :one
INSERT INTO people (name) VALUES (?) RETURNING id;

-- name: GetAllPeople :many
SELECT id, name FROM people;

-- name: InsertBorrowing :exec
INSERT INTO borrowing (isbn, person_id, borrowed_at) VALUES (?, ?, datetime('now'));

-- name: GetActiveBorrowings :many
SELECT b.id, b.isbn, b.person_id, b.borrowed_at, p.name as person_name
FROM borrowing b
JOIN people p ON b.person_id = p.id
WHERE b.returned_at IS NULL;

-- name: ReturnBook :exec
UPDATE borrowing 
SET returned_at = datetime('now') 
WHERE isbn = ? AND person_id = ? AND returned_at IS NULL;