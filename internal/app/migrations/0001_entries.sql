CREATE TABLE entries (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    spelling TEXT NOT NULL,
    spelling_key TEXT NOT NULL UNIQUE,
    definition TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
