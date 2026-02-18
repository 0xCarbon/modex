# Modern Go Guide (2026)

A comprehensive reference for writing idiomatic Go targeting **Go 1.25+** with `go1.26.0` toolchain.

---

## Tool Usage Workflow

**Always follow this sequence when working on a Go project with modex.**

### 1. Check the project's Go version first

```
get_go_version_info(project_path=".")
```

This tells you which language features and standard library APIs are available. Do not assume; the `go` directive in `go.mod` is the ground truth. Features gated on Go 1.24+ are unavailable if the module declares `go 1.21`.

### 2. Verify packages before importing

```
search_docs(query="strings.CutPrefix", project_path=".")
```

Call `search_docs` before writing any `import` for an unfamiliar package or API. This prevents dependency hallucination and confirms the exact function signature. Never import a package based on memory alone.

### 3. Run diagnostics after writing code

```
get_diagnostics(project_path=".")
```

This runs `go build`, `go vet`, and (if applicable) the race detector. Fix all reported issues before moving on. Go's compiler is strict — unused imports and undefined variables are hard errors, not warnings.

### 4. Scan for stale idioms

```
get_modernize_diagnostics(project_path=".")
```

This runs `go tool fix` across the module and reports patterns that can be replaced with modern equivalents. Apply all suggested fixes. The modernizer table below maps each analyzer to its version gate.

---

## Go Version Feature Matrix

Use the `go` directive in `go.mod` to determine which row applies to the project.

| Version | Key additions |
|---------|--------------|
| 1.18 | `any` alias for `interface{}`; `strings.Cut`; generics |
| 1.19 | `fmt.Appendf` / `fmt.Append` / `fmt.Appendln` |
| 1.20 | `strings.CutPrefix` / `CutSuffix`; `errors.Join`; `slices` / `maps` (x/exp graduates to proposal) |
| 1.21 | `min` / `max` builtins; `slices`, `maps`, `cmp` in stdlib; `slog` structured logging |
| 1.22 | Range-over-int (`for i := range n`); per-iteration loop variable (no more `x := x`); `reflect.TypeFor[T]()` |
| 1.23 | Range-over-func iterators; `maps.Insert`, `maps.Collect`; `slices.Collect`; `bytes`/`strings` `*Seq` iterators |
| 1.24 | `t.Context()` in tests; `strings.SplitSeq` / `FieldsSeq`; `json:",omitzero"` tag; `testing/synctest` (experimental → stable in 1.25) |
| 1.25 | `sync.WaitGroup.Go()`; `testing/synctest` stable; `runtime/trace.FlightRecorder`; `net/http.CrossOriginProtection`; `go vet` `waitgroup` + `hostport` analyzers |
| 1.26 | `new(expr)` builtin; `errors.AsType[T]` / `errors.IsType[T]`; `//go:fix inline` directive; `go tool fix` ships all 22 analyzers |

---

## Complete Modernizer Table

All 22 analyzers available in `go tool fix` as of `go1.26.0`. Run `go tool fix -fix ./...` to apply all applicable fixes at once, or `go tool fix -NAME -fix ./...` to apply selectively.

### String operations

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `stringscut` | 1.18 | `strings.Index(s,sep)` + manual slice | `strings.Cut(s, sep)` |
| `stringscutprefix` | 1.20 | `strings.HasPrefix` + `strings.TrimPrefix` | `strings.CutPrefix(s, prefix)` |
| `stringsbuilder` | 1.10 | `s += x` repeated in a loop | `strings.Builder` |
| `stringsseq` | 1.24 | `for _, s := range strings.Split(...)` | `for s := range strings.SplitSeq(...)` |

### Slice and map operations

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `slicessort` | 1.21 | `sort.Slice(s, func(i,j int) bool { return s[i] < s[j] })` | `slices.Sort(s)` |
| `slicescontains` | 1.21 | Manual loop to check element existence | `slices.Contains(s, v)` or `slices.ContainsFunc` |
| `mapsloop` | 1.23 | `for k, v := range src { dst[k] = v }` | `maps.Copy(dst, src)` / `maps.Clone(src)` |
| `stditerators` | 1.23 | `for i := 0; i < x.Len(); i++ { x.At(i) }` | `for v := range x.All()` |

### Concurrency

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `waitgroup` | 1.25 | `wg.Add(1); go func() { defer wg.Done(); ... }()` | `wg.Go(func() { ... })` |

### Testing

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `testingcontext` | 1.24 | `ctx, cancel := context.WithCancel(context.Background()); defer cancel()` in tests | `ctx := t.Context()` |

### Formatting and I/O

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `fmtappendf` | 1.19 | `[]byte(fmt.Sprintf(...))` | `fmt.Appendf(nil, ...)` |
| `hostport` | 1.0 | `fmt.Sprintf("%s:%d", host, port)` (breaks IPv6) | `net.JoinHostPort(host, strconv.Itoa(port))` |
| `minmax` | 1.21 | `if a < b { x = a } else { x = b }` | `x = min(a, b)` |

### Reflection

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `reflecttypefor` | 1.22 | `reflect.TypeOf((*T)(nil)).Elem()` | `reflect.TypeFor[T]()` |

### Loop variables

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `rangeint` | 1.22 | `for i := 0; i < n; i++ { ... }` | `for i := range n { ... }` |
| `forvar` | 1.22 | `for _, x := range s { x := x; ... }` (old capture workaround) | Remove the `x := x` line; Go 1.22+ captures per iteration |

### JSON encoding

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `omitzero` | 1.24 | `json:",omitempty"` on struct-typed fields (has no effect) | `json:",omitzero"` |

### Build tags

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `plusbuild` | 1.17 | Obsolete `// +build linux` comment when `//go:build` is also present | Remove the old `// +build` line |
| `buildtag` | 1.17 | Malformed or inconsistent `//go:build` / `// +build` pairs | Fix syntax; use only `//go:build` |

### Code structure

| Analyzer | Min Go | Replaces | Use instead |
|----------|--------|---------|-------------|
| `newexpr` | 1.26 | `func ptrOf(x T) *T { return &x }` helper functions | `new(x)` directly at call site |
| `inline` | 1.26 | Calls to functions/constants marked `//go:fix inline` | Inlined body (applied automatically) |

---

## Concurrency Safety

Concurrency is the #1 source of LLM-generated Go bugs. Every pattern below has caused real production incidents.

### WaitGroup — always use `wg.Go()` (Go 1.25+)

**WRONG — `Add` inside goroutine races with `Wait`:**

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)           // BAD: Add and Wait can race
    go func(v Item) {
        defer wg.Done()
        process(v)
    }(item)
}
wg.Wait()
```

**WRONG — old capture workaround (pre-1.22 pattern, no longer needed):**

```go
for _, item := range items {
    item := item           // BAD: unnecessary since Go 1.22
    wg.Add(1)
    go func() {
        defer wg.Done()
        process(item)
    }()
}
```

**CORRECT — `wg.Go()` (Go 1.25+):**

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Go(func() {
        process(item)   // item captured correctly per-iteration (Go 1.22+)
    })
}
wg.Wait()
```

**CORRECT — if you must support Go < 1.25, call `Add` before launching:**

```go
var wg sync.WaitGroup
for _, item := range items {
    wg.Add(1)           // Add BEFORE the goroutine is launched
    go func() {
        defer wg.Done()
        process(item)
    }()
}
wg.Wait()
```

`go vet` (via the `waitgroup` analyzer) flags the `Add`-inside-goroutine pattern.

### Channel deadlocks

**WRONG — send on unbuffered channel with no receiver:**

```go
ch := make(chan int)
ch <- 42            // blocks forever; nobody is receiving
v := <-ch
```

**WRONG — goroutine leak with no cancellation:**

```go
func fetch(url string) <-chan Result {
    ch := make(chan Result)
    go func() {
        ch <- doFetch(url)  // leaks if caller never reads
    }()
    return ch
}
```

**CORRECT — use context for cancellation:**

```go
func fetch(ctx context.Context, url string) <-chan Result {
    ch := make(chan Result, 1)
    go func() {
        select {
        case ch <- doFetch(url):
        case <-ctx.Done():
        }
    }()
    return ch
}
```

**CORRECT — fan-out with errgroup:**

```go
import "golang.org/x/sync/errgroup"

g, ctx := errgroup.WithContext(context.Background())
for _, url := range urls {
    g.Go(func() error {
        return process(ctx, url)
    })
}
if err := g.Wait(); err != nil {
    return err
}
```

### Goroutine leak prevention

Every goroutine needs an exit path. Without one, the goroutine runs until the process dies.

**WRONG — goroutine with no shutdown:**

```go
func startWorker() {
    go func() {
        for {
            doWork()    // runs forever, no way to stop it
        }
    }()
}
```

**CORRECT — context-driven shutdown:**

```go
func startWorker(ctx context.Context) {
    go func() {
        for {
            select {
            case <-ctx.Done():
                return
            default:
                doWork()
            }
        }
    }()
}
```

**In tests, use `t.Context()` (Go 1.24+)** instead of creating a context manually:

```go
// WRONG (before Go 1.24):
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// CORRECT (Go 1.24+):
ctx := t.Context()  // automatically cancelled when the test ends
```

### Map races

Maps are not safe for concurrent read/write. This is a data race, detected by `-race`.

**WRONG:**

```go
m := make(map[string]int)
go func() { m["a"] = 1 }()
go func() { _ = m["b"] }()  // concurrent read+write = undefined behavior
```

**CORRECT — `sync.Map` for concurrent access:**

```go
var m sync.Map
go func() { m.Store("a", 1) }()
go func() {
    if v, ok := m.Load("b"); ok {
        use(v)
    }
}()
```

**CORRECT — mutex-protected map (preferred when reads dominate):**

```go
type SafeMap struct {
    mu sync.RWMutex
    m  map[string]int
}

func (s *SafeMap) Set(k string, v int) {
    s.mu.Lock()
    defer s.mu.Unlock()
    s.m[k] = v
}

func (s *SafeMap) Get(k string) (int, bool) {
    s.mu.RLock()
    defer s.mu.RUnlock()
    v, ok := s.m[k]
    return v, ok
}
```

### sync.Cond — prefer channels

`sync.Cond` with `Signal` (not `Broadcast`) deadlocks when both producers and consumers wait on the same condition. LLM-generated `sync.Cond` code reliably fails under high concurrency.

**WRONG — single Cond, wrong waiter woken:**

```go
// Producer signals, but may wake another producer instead of the consumer
cond.Signal()
```

**CORRECT — prefer channels for producer/consumer:**

```go
work := make(chan Item, 64)
results := make(chan Result, 64)

go func() {       // producer
    for _, item := range items {
        work <- item
    }
    close(work)
}()

go func() {       // consumer
    for item := range work {
        results <- process(item)
    }
    close(results)
}()
```

If `sync.Cond` is unavoidable, use `Broadcast` (wakes all waiters) and re-check the condition in a loop:

```go
cond.L.Lock()
for !ready() {
    cond.Wait()
}
cond.L.Unlock()
```

### testing/synctest — deterministic concurrency tests (Go 1.25+)

`testing/synctest` runs a test in an isolated bubble with a fake clock. All goroutines started inside the bubble are tracked; `synctest.Wait()` blocks until they all quiesce.

```go
import "testing/synctest"

func TestConcurrentCache(t *testing.T) {
    synctest.Run(func() {
        cache := NewCache()
        var wg sync.WaitGroup
        wg.Go(func() { cache.Set("k", 1) })
        wg.Go(func() { cache.Set("k", 2) })
        wg.Wait()
        synctest.Wait()  // all goroutines have settled
    })
}
```

### Race detector — mandatory for concurrent code

Always run with `-race` after writing or modifying concurrent code:

```bash
go test -race ./...
go build -race -o myapp_race .
```

The race detector has near-zero false positives. A race report means there is a real bug.

---

## GODEBUG Awareness

Bumping the `go` directive in `go.mod` silently changes runtime behavior via GODEBUG defaults. This is the most common source of "it worked before the upgrade" bugs.

### Loop variable scoping (Go 1.22)

Before Go 1.22, all iterations of a `for` loop shared one loop variable. Goroutines launched inside the loop captured the variable, not the value.

**Before Go 1.22 — required the capture workaround:**

```go
for _, v := range items {
    v := v              // capture by value; REQUIRED before 1.22
    go func() { use(v) }()
}
```

**Go 1.22+ — each iteration gets its own variable:**

```go
for _, v := range items {
    // no copy needed; v is fresh each iteration
    go func() { use(v) }()
}
```

When upgrading a module from `go 1.21` to `go 1.22`, run `go fix -forvar ./...` to remove the now-redundant `x := x` lines.

### Controlling GODEBUG during migration

Pin specific GODEBUG values in `go.mod` to opt out of behavior changes while testing:

```
module example.com/myapp

go 1.22

godebug loopvar=0  // revert to pre-1.22 loop semantics during migration
```

Remove the `godebug` directive once migration is complete and tests pass. Full list of per-version GODEBUG defaults: `go doc runtime/internal/sys GODEBUG`.

### Common GODEBUG keys

| Key | Changed in | Behavior |
|-----|-----------|---------|
| `loopvar` | 1.22 | Per-iteration loop variables (`loopvar=1` is the 1.22+ default) |
| `randseekpos` | 1.20 | `math/rand` global source seeding |
| `tlsrsakey` | 1.23 | TLS RSA key size minimum |
| `httpmuxgo121` | 1.22 | Old `net/http` ServeMux behavior |

---

## Dependency Hallucination Prevention

LLMs generate fictitious Go package paths at a measured rate of ~20% (USENIX Security 2025). Go's module system makes hallucinated paths detectable but not harmless — they waste time and can introduce supply-chain risk.

### Rule: verify before importing

Before writing any `import` for a package you haven't verified:

1. Call `search_docs(query="<package or function>", project_path=".")` to confirm it exists.
2. If `search_docs` finds nothing, search `pkg.go.dev` manually before adding it to `go.mod`.
3. Never invent a module path. If you're unsure, use the stdlib alternative.

### Stdlib covers most needs

| Task | Use | Not |
|------|-----|-----|
| HTTP client / server | `net/http` | `resty`, `fasthttp` (unless benchmarks justify it) |
| JSON | `encoding/json` | `jsoniter`, `go-json` (unless allocation matters) |
| Structured logging | `log/slog` (1.21+) | `logrus`, `zap` (unless already in use) |
| String manipulation | `strings`, `unicode` | `github.com/huandu/xstrings` |
| Slices | `slices` (1.21+) | `github.com/samber/lo` |
| Maps | `maps` (1.21+) | custom utilities |
| Context / cancellation | `context` | any third-party context package |
| Testing FS | `testing/fstest` | custom interfaces |
| Error wrapping | `errors`, `fmt.Errorf` | `github.com/pkg/errors` (unmaintained) |

### Dependency hygiene

```bash
go mod tidy           # remove unused, add missing
go mod verify         # check that downloaded modules match go.sum
```

Treat every new entry in `go.mod` as a security-sensitive change. Review `go.sum` diffs in code review.

### Commonly confused packages

| What LLMs often suggest | What to use instead | Why |
|------------------------|-------------------|-----|
| `github.com/pkg/errors` | stdlib `errors` + `fmt.Errorf("%w", err)` | `pkg/errors` is archived |
| `github.com/golang/protobuf` | `google.golang.org/protobuf` | Old module, use the new one |
| `gopkg.in/yaml.v2` | `gopkg.in/yaml.v3` | v2 is unmaintained |
| `ioutil.ReadAll` | `io.ReadAll` (1.16+) | `ioutil` deprecated in 1.16 |
| `ioutil.WriteFile` | `os.WriteFile` (1.16+) | Same reason |
| `ioutil.TempFile` | `os.CreateTemp` (1.16+) | Same reason |

---

## Error Handling

### Wrap with `%w` at every layer

```go
func loadConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("loadConfig: %w", err)
    }
    var cfg Config
    if err := json.Unmarshal(data, &cfg); err != nil {
        return nil, fmt.Errorf("loadConfig: parse %s: %w", path, err)
    }
    return &cfg, nil
}
```

Always use `%w` (not `%s` or `%v`) when wrapping — it preserves the error chain for `errors.Is` / `errors.As`.

### Sentinel errors

```go
var (
    ErrNotFound = errors.New("not found")
    ErrConflict = errors.New("conflict")
)

// caller:
if errors.Is(err, ErrNotFound) {
    http.Error(w, "not found", http.StatusNotFound)
}
```

### Multiple errors — `errors.Join` (Go 1.20+)

```go
func validateAll(fields []Field) error {
    var errs []error
    for _, f := range fields {
        if err := f.Validate(); err != nil {
            errs = append(errs, err)
        }
    }
    return errors.Join(errs...)
}
```

### Type-checked unwrapping — `errors.AsType` / `errors.IsType` (Go 1.26+)

```go
// Before 1.26:
var target *pgconn.PgError
if errors.As(err, &target) && target.Code == "23505" { ... }

// Go 1.26+:
if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" { ... }
```

### Do not ignore errors

```go
// WRONG:
os.Remove(tmpFile)

// CORRECT:
if err := os.Remove(tmpFile); err != nil && !errors.Is(err, os.ErrNotExist) {
    log.Warn("failed to remove temp file", "path", tmpFile, "err", err)
}
```

---

## Standard Library — Blessed Packages

### `slices` (Go 1.21+)

Prefer `slices` over manual loops. All functions work on any `[]E` where `E` is comparable or ordered.

```go
import "slices"

slices.Contains(s, v)           // linear search
slices.ContainsFunc(s, pred)    // predicate search
slices.Sort(s)                  // in-place sort (ordered types)
slices.SortFunc(s, cmp)         // custom comparator
slices.Index(s, v)              // first index of v, or -1
slices.Compact(s)               // remove consecutive duplicates
slices.Reverse(s)               // in-place reverse
slices.Clone(s)                 // shallow copy
slices.Collect(iter)            // iterator → slice (Go 1.23+)
```

### `maps` (Go 1.21+)

```go
import "maps"

maps.Keys(m)                    // iterator over keys (Go 1.23+)
maps.Values(m)                  // iterator over values (Go 1.23+)
maps.Clone(m)                   // shallow copy
maps.Copy(dst, src)             // merge src into dst
maps.Insert(dst, iter)          // insert from iterator (Go 1.23+)
maps.Collect(iter)              // iterator → map (Go 1.23+)
maps.Delete(m, k)               // delete key (no-op if absent)
```

### `cmp` (Go 1.21+)

```go
import "cmp"

cmp.Compare(a, b)   // -1, 0, +1 for ordered types
cmp.Or(a, b, c)     // first non-zero value; useful for defaults
```

### `log/slog` (Go 1.21+) — replace `log.Printf`

```go
import "log/slog"

// basic usage
slog.Info("server started", "addr", addr, "pid", os.Getpid())
slog.Error("query failed", "err", err, "query", q)

// structured logger with JSON output
logger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
    Level: slog.LevelInfo,
}))
logger.Info("request", "method", r.Method, "path", r.URL.Path)
```

Do not use `log.Printf` in new code. `slog` is structured, level-aware, and composable.

### `encoding/json` — tag guidance

```go
type User struct {
    Name      string    `json:"name"`
    CreatedAt time.Time `json:"created_at,omitempty"`  // omits zero time.Time? NO — use omitzero
    Address   *Address  `json:"address,omitempty"`     // pointer: omits nil correctly
    Score     int       `json:"score,omitempty"`       // primitive: omits zero value
    Profile   Profile   `json:"profile,omitzero"`      // struct: omits zero-value struct (Go 1.24+)
}
```

`omitempty` on a struct-typed field has no effect. Use `omitzero` (Go 1.24+) instead, or make the field a pointer.

---

## Testing Patterns

### Table-driven tests (canonical Go pattern)

```go
func TestAdd(t *testing.T) {
    cases := []struct {
        name string
        a, b int
        want int
    }{
        {"positive", 1, 2, 3},
        {"zero", 0, 0, 0},
        {"negative", -1, -2, -3},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            t.Parallel()
            got := Add(tc.a, tc.b)
            if got != tc.want {
                t.Errorf("Add(%d, %d) = %d; want %d", tc.a, tc.b, got, tc.want)
            }
        })
    }
}
```

`t.Parallel()` inside `t.Run` makes subtests run concurrently. Always add it to independent subtests.

### Context in tests — `t.Context()` (Go 1.24+)

```go
// WRONG (verbose, cancel might be forgotten):
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

// CORRECT (Go 1.24+):
ctx := t.Context()  // cancelled automatically when t.Cleanup runs
```

### Filesystem tests — `testing/fstest.MapFS`

```go
import "testing/fstest"

fs := testing/fstest.MapFS{
    "config.yaml": {Data: []byte("key: value")},
    "data/item.txt": {Data: []byte("hello")},
}
cfg, err := loadConfig(fs, "config.yaml")
```

### Concurrent code — `testing/synctest` (Go 1.25+)

```go
import "testing/synctest"

func TestWorkerDrainsQueue(t *testing.T) {
    synctest.Run(func() {
        q := NewQueue()
        q.Enqueue("a")
        q.Enqueue("b")

        var wg sync.WaitGroup
        wg.Go(func() { q.Process() })
        synctest.Wait()   // all goroutines in bubble have quiesced

        wg.Wait()
        if q.Len() != 0 {
            t.Error("queue not drained")
        }
    })
}
```

### Race detection

Always run with `-race` for any package with goroutines:

```bash
go test -race ./...
```

### Black-box test packaging rule

`package foo_test` (external test package) **cannot** access unexported identifiers from `package foo`. Use `package foo` for tests that need unexported access.

```go
// file: parser_test.go

// WRONG if parseInternal is unexported:
package parser_test
import "example.com/myapp/parser"
func TestInternal(t *testing.T) {
    parser.parseInternal("x")  // compile error: unexported
}

// CORRECT: same package for internal tests
package parser
func TestInternal(t *testing.T) {
    parseInternal("x")  // OK
}
```

---

## Project Setup

### `go.mod` structure

```
module github.com/yourorg/yourapp

go 1.25

toolchain go1.26.0

require (
    golang.org/x/sync v0.11.0
)
```

- `go` directive: set to the **minimum** Go version your code actually requires, not the latest.
- `toolchain` directive: pin the specific toolchain used for development; teammates get the same version.
- Run `go mod tidy` before every commit.

### GODEBUG migration

```
module github.com/yourorg/yourapp

go 1.22

godebug loopvar=0   // remove once all loops are tested under 1.22 semantics
```

### Build tags — `//go:build` only

```go
//go:build linux && amd64

package main
```

Never write the old `// +build` form in new code. The `plusbuild` and `buildtag` modernizers will flag stale build constraints.

### Directory conventions

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go       // entry point per binary
├── internal/             // packages not exported to other modules
│   ├── config/
│   └── store/
├── go.mod
└── go.sum
```

Do not create a top-level `src/` directory — that is not Go convention.

---

## Anti-Patterns

### `interface{}` → `any`

```go
// WRONG:
func Print(v interface{}) { fmt.Println(v) }

// CORRECT:
func Print(v any) { fmt.Println(v) }
```

### `sort.Slice` for basic ordered types → `slices.Sort`

```go
// WRONG:
sort.Slice(s, func(i, j int) bool { return s[i] < s[j] })

// CORRECT:
slices.Sort(s)
```

### String concatenation in loops → `strings.Builder`

```go
// WRONG — O(n²) allocations:
var result string
for _, item := range items {
    result += item + ", "
}

// CORRECT:
var b strings.Builder
for _, item := range items {
    b.WriteString(item)
    b.WriteString(", ")
}
result := b.String()
```

### Network address formatting → `net.JoinHostPort`

```go
// WRONG — breaks IPv6 addresses:
addr := fmt.Sprintf("%s:%d", host, port)

// CORRECT:
addr := net.JoinHostPort(host, strconv.Itoa(port))
```

### Byte slice formatting → `fmt.Appendf`

```go
// WRONG — allocates a string then converts:
b := []byte(fmt.Sprintf("value=%d", n))

// CORRECT:
b := fmt.Appendf(nil, "value=%d", n)
```

### Prefix check + trim → `strings.CutPrefix`

```go
// WRONG:
if strings.HasPrefix(s, "Bearer ") {
    token := strings.TrimPrefix(s, "Bearer ")
    use(token)
}

// CORRECT:
if token, ok := strings.CutPrefix(s, "Bearer "); ok {
    use(token)
}
```

### 3-clause for loop over integers → range-over-int

```go
// WRONG:
for i := 0; i < n; i++ {
    process(i)
}

// CORRECT (Go 1.22+):
for i := range n {
    process(i)
}
```

### Manual context in tests → `t.Context()`

```go
// WRONG:
ctx, cancel := context.WithCancel(context.Background())
defer cancel()
result, err := svc.Call(ctx, req)

// CORRECT (Go 1.24+):
result, err := svc.Call(t.Context(), req)
```

### WaitGroup boilerplate → `wg.Go()`

```go
// WRONG:
wg.Add(1)
go func() {
    defer wg.Done()
    process(item)
}()

// CORRECT (Go 1.25+):
wg.Go(func() {
    process(item)
})
```

### `reflect.TypeOf` zero value → `reflect.TypeFor`

```go
// WRONG:
t := reflect.TypeOf((*io.Reader)(nil)).Elem()

// CORRECT (Go 1.22+):
t := reflect.TypeFor[io.Reader]()
```

### Ioutil functions (deprecated since 1.16)

```go
// WRONG:
data, err := ioutil.ReadAll(r)
err = ioutil.WriteFile(path, data, 0644)
f, err := ioutil.TempFile("", "prefix")

// CORRECT:
data, err := io.ReadAll(r)
err = os.WriteFile(path, data, 0644)
f, err := os.CreateTemp("", "prefix")
```

---

## Concurrency Primitives Reference

| Need | Use |
|------|-----|
| Multiple goroutines, collect errors | `golang.org/x/sync/errgroup` |
| Goroutine with return value | `errgroup.Group` or channel with buffer 1 |
| Shared mutable state | `sync.Mutex` (short critical sections) |
| Read-heavy shared state | `sync.RWMutex` |
| Concurrent map access | `sync.Map` or `sync.RWMutex`-protected map |
| One-time initialization | `sync.Once` |
| Atomic counters / flags | `sync/atomic` (`atomic.Int64`, `atomic.Bool`) |
| Goroutine fan-out + collect | `errgroup` or `sync.WaitGroup` + channel |
| Rate limiting | `golang.org/x/time/rate` |
| Bounded parallelism | `errgroup.WithContext` + semaphore channel |
| Publish/subscribe | `golang.org/x/sync/singleflight` or buffered channels |

Use `context.Context` as the first parameter of any function that may block or launch goroutines. Propagate it through the entire call chain.
