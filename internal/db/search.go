package db

import (
	"fmt"
	"strings"
)

// SearchResult holds one documentable symbol returned by SearchDocs.
type SearchResult struct {
	PackagePath   string `json:"package_path"`
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

// SearchParams holds all search configuration for SearchDocs.
type SearchParams struct {
	// Query is the search string; required.
	Query string
	// Mode controls how Query is interpreted: "auto", "text", "symbol", "fts5".
	// Default (empty string) is treated as "auto".
	Mode string
	// Packages restricts results to these package paths (exact prefix match).
	Packages []string
	// Kinds restricts results to these symbol kinds (e.g. "func", "type", "method").
	Kinds []string
	// Parent filters by parent type name (e.g. "Reader" to find Reader methods).
	Parent string
	// Limit is the maximum number of results (default 10, max 50).
	Limit int
	// Offset is the pagination offset.
	Offset int
}

// SearchResponse holds results and a pagination indicator.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	HasMore bool           `json:"has_more"`
}

// SearchDocs queries the FTS5 index with all provided filters, returning ranked results.
func (db *DB) SearchDocs(p SearchParams) (SearchResponse, error) {
	if p.Limit <= 0 {
		p.Limit = 10
	}
	if p.Limit > 50 {
		p.Limit = 50
	}

	ftsQuery := buildFTSQuery(p.Query, p.Mode)
	if ftsQuery == "" {
		return SearchResponse{}, nil
	}

	// Fetch limit+1 to detect if there are more results.
	fetch := p.Limit + 1

	query, args := buildSearchQuery(ftsQuery, p, fetch)
	rows, err := db.Query(query, args...)
	if err != nil {
		return SearchResponse{}, fmt.Errorf("search: %w", err)
	}
	defer rows.Close()

	var items []SearchResult
	for rows.Next() {
		var it SearchResult
		if err := rows.Scan(
			&it.PackagePath, &it.ModulePath, &it.ModuleVersion, &it.GoVersion,
			&it.ItemName, &it.Kind,
			&it.ParentName, &it.ParentKind,
			&it.Signature, &it.DocText,
		); err != nil {
			return SearchResponse{}, fmt.Errorf("search scan: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return SearchResponse{}, err
	}

	hasMore := len(items) > p.Limit
	if hasMore {
		items = items[:p.Limit]
	}
	return SearchResponse{Results: items, HasMore: hasMore}, nil
}

// buildFTSQuery converts a user query + mode into an FTS5 MATCH expression.
func buildFTSQuery(q, mode string) string {
	q = strings.TrimSpace(q)
	if q == "" {
		return ""
	}
	switch mode {
	case "fts5":
		// Power user: pass raw FTS5 syntax directly.
		return q
	case "symbol":
		return symbolQuery(q)
	case "text":
		return textQuery(q)
	default: // "auto" or ""
		if strings.ContainsAny(q, ".:/") {
			return symbolQuery(q)
		}
		return textQuery(q)
	}
}

// textQuery builds a prefix-match FTS5 query from space-separated tokens.
// "read all" → "read* AND all*"
func textQuery(q string) string {
	tokens := strings.Fields(q)
	if len(tokens) == 0 {
		return ""
	}
	escaped := make([]string, len(tokens))
	for i, t := range tokens {
		escaped[i] = ftsEscape(t) + "*"
	}
	return strings.Join(escaped, " AND ")
}

// symbolQuery handles "pkg.Symbol" and "Type.Method" patterns.
// "fmt.Println" → item_name:Println AND package_path:fmt*
// "io.Reader"   → item_name:Reader AND package_path:io*
// "Println"     → item_name:Println*
func symbolQuery(q string) string {
	// Strip leading import-path segments to get "pkg.Symbol" at minimum.
	pkg, sym, hasDot := strings.Cut(q, ".")
	if !hasDot {
		// No dot: treat as item name prefix.
		return "item_name:" + ftsEscape(q) + "*"
	}
	return fmt.Sprintf("item_name:%s* AND package_path:%s*",
		ftsEscape(sym), ftsEscape(pkg))
}

// ftsEscape wraps a token in double quotes if it contains FTS5 special chars.
func ftsEscape(s string) string {
	const special = " \t\"'*():"
	if strings.ContainsAny(s, special) {
		return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
	}
	return s
}

// buildSearchQuery constructs the parameterised SQL query and argument list.
func buildSearchQuery(ftsQuery string, p SearchParams, fetch int) (string, []any) {
	var sb strings.Builder
	var args []any

	sb.WriteString(`SELECT d.package_path, d.module_path, d.module_version,
       d.go_version, d.item_name, d.kind,
       COALESCE(d.parent_name,''), COALESCE(d.parent_kind,''),
       d.signature, d.doc_text
FROM docs d
JOIN docs_fts f ON f.rowid = d.id
WHERE docs_fts MATCH ?`)
	args = append(args, ftsQuery)

	if len(p.Packages) > 0 {
		placeholders := strings.Repeat("?,", len(p.Packages))
		placeholders = placeholders[:len(placeholders)-1]
		sb.WriteString(" AND d.package_path IN (" + placeholders + ")")
		for _, pkg := range p.Packages {
			args = append(args, pkg)
		}
	}

	if len(p.Kinds) > 0 {
		placeholders := strings.Repeat("?,", len(p.Kinds))
		placeholders = placeholders[:len(placeholders)-1]
		sb.WriteString(" AND d.kind IN (" + placeholders + ")")
		for _, k := range p.Kinds {
			args = append(args, k)
		}
	}

	if p.Parent != "" {
		sb.WriteString(" AND d.parent_name = ?")
		args = append(args, p.Parent)
	}

	// BM25 weights: package_path=0, item_name=3, parent_name=2, signature=1.5, doc_text=1
	sb.WriteString(" ORDER BY bm25(docs_fts, 0, 3, 2, 1.5, 1) LIMIT ? OFFSET ?")
	args = append(args, fetch, p.Offset)

	return sb.String(), args
}
