-- name: GetBook :one
SELECT isbn, title, description, publisher, published_date, pages, language, cover_url, row_number, shelf_id 
FROM books 
WHERE isbn = ?;

-- name: GetAllBooks :many
SELECT isbn, title, description, publisher, published_date, pages, language, cover_url, shelf_id, row_number 
FROM books;

-- name: InsertBook :exec
INSERT INTO books 
(isbn, title, description, publisher, published_date, pages, language, cover_url, row_number, shelf_id)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(isbn) DO UPDATE SET
    row_number = excluded.row_number,
    shelf_id = excluded.shelf_id;

-- name: DeleteBook :execrows
DELETE FROM books WHERE isbn = ?;

-- name: GetUnenrichedBooks :many
SELECT isbn FROM books WHERE is_ai_enriched = 0;

-- name: UpdateBookTitle :exec
UPDATE books SET title = ? WHERE isbn = ? AND (title IS NULL OR title = '');

-- name: UpdateBookDescription :exec
UPDATE books SET description = ? WHERE isbn = ? AND (description IS NULL OR description = '');

-- name: UpdateBookPublishedDate :exec
UPDATE books SET published_date = ? WHERE isbn = ? AND (published_date IS NULL OR published_date = '');

-- name: MarkBookAsEnriched :exec
UPDATE books SET is_ai_enriched = 1 WHERE isbn = ?;