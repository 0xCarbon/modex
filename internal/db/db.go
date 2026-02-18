// Package db provides a SQLite wrapper with FTS5 full-text search for Go documentation.
package db

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver.
)

const schema = `
PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS docs (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    package_path   TEXT NOT NULL,
    package_hash   TEXT NOT NULL,
    module_path    TEXT NOT NULL DEFAULT '',
    module_version TEXT NOT NULL DEFAULT '',
    go_version     TEXT NOT NULL DEFAULT '',
    item_name      TEXT NOT NULL,
    kind           TEXT NOT NULL,
    parent_name    TEXT,
    parent_kind    TEXT,
    signature      TEXT NOT NULL DEFAULT '',
    doc_text       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_docs_package_hash ON docs (package_hash);
CREATE INDEX IF NOT EXISTS idx_docs_package_path ON docs (package_path);
CREATE INDEX IF NOT EXISTS idx_docs_parent ON docs (parent_name);

CREATE VIRTUAL TABLE IF NOT EXISTS docs_fts USING fts5(
    package_path, item_name, parent_name, signature, doc_text,
    content='docs', content_rowid='id',
    tokenize='porter unicode61'
);

-- Auto-sync triggers: keep FTS5 in sync with the docs table.
CREATE TRIGGER IF NOT EXISTS docs_ai AFTER INSERT ON docs BEGIN
    INSERT INTO docs_fts(rowid, package_path, item_name, parent_name, signature, doc_text)
    VALUES (new.id, new.package_path, new.item_name,
            COALESCE(new.parent_name, ''), new.signature, new.doc_text);
END;

CREATE TRIGGER IF NOT EXISTS docs_ad AFTER DELETE ON docs BEGIN
    INSERT INTO docs_fts(docs_fts, rowid, package_path, item_name, parent_name, signature, doc_text)
    VALUES ('delete', old.id, old.package_path, old.item_name,
            COALESCE(old.parent_name, ''), old.signature, old.doc_text);
END;

CREATE TRIGGER IF NOT EXISTS docs_au AFTER UPDATE ON docs BEGIN
    INSERT INTO docs_fts(docs_fts, rowid, package_path, item_name, parent_name, signature, doc_text)
    VALUES ('delete', old.id, old.package_path, old.item_name,
            COALESCE(old.parent_name, ''), old.signature, old.doc_text);
    INSERT INTO docs_fts(rowid, package_path, item_name, parent_name, signature, doc_text)
    VALUES (new.id, new.package_path, new.item_name,
            COALESCE(new.parent_name, ''), new.signature, new.doc_text);
END;
`

// DB wraps a *sql.DB with modex-specific schema and helpers.
type DB struct {
	*sql.DB
}

// SearchResult holds one documentable symbol returned by Search.
type SearchResult struct {
	PackagePath   string `json:"package_path"`
	PackageHash   string `json:"package_hash"`
	ModulePath    string `json:"module_path"`
	ModuleVersion string `json:"module_version"`
	GoVersion     string `json:"go_version"`
	ItemName      string `json:"item_name"`
	Kind          string `json:"kind"`
	ParentName    string `json:"parent_name,omitempty"`
	ParentKind    string `json:"parent_kind,omitempty"`
	Signature     string `json:"signature"`
	DocText       string `json:"doc_text"`
}

// Open creates or opens a SQLite database at path, sets WAL mode, and runs the schema DDL.
// Use ":memory:" for an in-memory database (useful for tests).
func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("db open: %w", err)
	}
	// Single connection to avoid SQLite locking issues.
	sqlDB.SetMaxOpenConns(1)

	if _, err := sqlDB.Exec(schema); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("db schema: %w", err)
	}
	return &DB{sqlDB}, nil
}

// HashExists reports whether a row with the given package_hash exists in docs.
func (db *DB) HashExists(hash string) (bool, error) {
	var exists bool
	err := db.QueryRow("SELECT EXISTS(SELECT 1 FROM docs WHERE package_hash = ?)", hash).Scan(&exists)
	return exists, err
}

// DeleteByHash removes all docs rows matching the given package_hash.
func (db *DB) DeleteByHash(hash string) (int64, error) {
	res, err := db.Exec("DELETE FROM docs WHERE package_hash = ?", hash)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Search queries the FTS5 index and returns up to limit matching results ranked by BM25.
// limit defaults to 20 if ≤ 0.
func (db *DB) Search(query string, limit int) ([]SearchResult, error) {
	if limit <= 0 {
		limit = 20
	}
	const q = `
SELECT d.package_path, d.package_hash, d.module_path, d.module_version, d.go_version,
       d.item_name, d.kind,
       COALESCE(d.parent_name, ''), COALESCE(d.parent_kind, ''),
       d.signature, d.doc_text
FROM docs d
JOIN docs_fts f ON f.rowid = d.id
WHERE docs_fts MATCH ?
ORDER BY rank
LIMIT ?`
	rows, err := db.Query(q, query, limit)
	if err != nil {
		return nil, fmt.Errorf("search query: %w", err)
	}
	defer rows.Close()

	var items []SearchResult
	for rows.Next() {
		var it SearchResult
		if err := rows.Scan(
			&it.PackagePath, &it.PackageHash, &it.ModulePath, &it.ModuleVersion, &it.GoVersion,
			&it.ItemName, &it.Kind,
			&it.ParentName, &it.ParentKind,
			&it.Signature, &it.DocText,
		); err != nil {
			return nil, fmt.Errorf("search scan: %w", err)
		}
		items = append(items, it)
	}
	return items, rows.Err()
}
