-- Schema for librascan database

-- Shelfs table
CREATE TABLE shelfs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    rows_count INTEGER,
    UNIQUE(name)
);

-- Books table (after migrations)
CREATE TABLE books (
    ISBN INTEGER PRIMARY KEY,
    title TEXT,
    description TEXT,
    publisher TEXT,
    published_date TEXT,
    pages INTEGER,
    language TEXT,
    cover_url TEXT,
    shelf_id INTEGER,
    row_number INTEGER,
    is_ai_enriched INTEGER DEFAULT 0,
    FOREIGN KEY(shelf_id) REFERENCES shelfs(id)
);

-- Authors table
CREATE TABLE authors (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    isbn INTEGER,
    UNIQUE(name, isbn),
    FOREIGN KEY(isbn) REFERENCES books(ISBN)
);

-- Categories table
CREATE TABLE categories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT,
    isbn INTEGER,
    UNIQUE(name, isbn),
    FOREIGN KEY(isbn) REFERENCES books(ISBN)
);

-- People table
CREATE TABLE people (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    UNIQUE(name)
);

-- Borrowing table
CREATE TABLE borrowing (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    isbn INTEGER NOT NULL,
    person_id INTEGER NOT NULL,
    borrowed_at TEXT NOT NULL,
    returned_at TEXT,
    FOREIGN KEY(isbn) REFERENCES books(ISBN),
    FOREIGN KEY(person_id) REFERENCES people(id)
);

-- Enable foreign keys
PRAGMA foreign_keys = ON;