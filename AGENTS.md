# AGENTS.md — m3uparser

Guidelines for AI coding agents working in this repository.

## Project Overview

Go CLI/Docker application that parses M3U playlist files, creates `.strm` file
libraries (organized by series, TV shows, movies, live TV, unsorted), and integrates
with Jellyfin and Threadfin media servers. Designed to run as a Kubernetes CronJob
(run once and exit).

**Entry point:** `cmd/m3uparser/main.go`

## Build / Run / Test Commands

### Build

```bash
go build -o m3uparser ./cmd/m3uparser
```

### Run (local)

```bash
M3U_URL="http://example.com/playlist.m3u" go run ./cmd/m3uparser
```

Environment variables must be set (see `m3uparser/m3uparser.env` for the full list).

### Run (Docker)

```bash
docker compose up --build
```

### Tests

```bash
# Run all tests:
go test ./...

# Run a single package's tests:
go test ./internal/entry/...

# Run a single test by name:
go test ./internal/entry/... -run TestClassifySeries -v

# Run with race detector:
go test -race ./...
```

### Linting / Formatting

No linter is configured beyond `go vet`. If adding one, prefer `golangci-lint`.

```bash
go vet ./...
gofmt -l .
```

### CI/CD

GitHub Actions workflow at `.github/workflows/docker-image.yaml`:
- Triggers on push to `main`, `ezpztv`, `ezpztv_threadfin` branches
- Reads `.docker_version` to decide whether to build
- Builds and pushes Docker images to Docker Hub (`xaque87/m3uparser`)

## Project Structure

```
cmd/
  m3uparser/
    main.go                   # Application entry point (load config → download → parse → write → sync → integrations)
internal/
  config/
    config.go                 # Typed Config struct loaded from environment variables
    config_test.go            # Config loading tests
  entry/
    entry.go                  # Entry type definition and constants (TypeSeries, TypeTV, TypeMovie, TypeLiveTV, TypeUnsorted)
    classifier.go             # Entry classification (series/TV/movie/live/unsorted) and metadata extraction
    classifier_test.go        # Classifier tests
    cleaner.go                # Value processing: replace terms, scrub headers, remove terms, exclude check
    cleaner_test.go           # Cleaner tests
  m3u/
    download.go               # M3U file downloading (HEAD check or bypass) and combining
    parser.go                 # M3U file parsing, #EXTINF line extraction, key-value pair parsing
    parser_test.go            # Parser tests
  strm/
    writer.go                 # .strm file writing, live TV playlist, directory sync, cleanup, counting
    writer_test.go            # Writer tests
  jellyfin/
    client.go                 # Jellyfin HTTP client (API key auth, ping, library refresh, guide refresh)
  threadfin/
    client.go                 # Threadfin REST API client (login/token, M3U/XMLTV/xEPG updates)
```

### Legacy Python code (still present, for reference only):

```
parser/                       # Entire Python source tree (not used by Go build)
entrypoint.sh                 # Python-era entrypoint
requirements.txt              # Python dependencies
```

## Code Style Guidelines

### Formatting

- **Use `gofmt`** — all Go code must be formatted with `gofmt`
- **Line length:** No hard limit, but keep under 120 chars where practical
- **Imports:** Group stdlib, then third-party, then local (`github.com/dani/m3uparser/...`),
  separated by blank lines

### Naming Conventions

| Element        | Convention     | Examples                                    |
|----------------|----------------|---------------------------------------------|
| Functions      | `PascalCase`   | `ClassifyAndClean`, `ParseFile`, `WriteAll` |
| Unexported     | `camelCase`    | `writeSeries`, `extractSeasonEpisode`       |
| Variables      | `camelCase`    | `m3uDir`, `liveTVCount`                     |
| Constants      | `PascalCase`   | `TypeSeries`, `TypeLiveTV`                  |
| Config fields  | `PascalCase`   | `M3UURLs`, `BypassHeader`, `CleanSync`      |
| Env vars       | `UPPER_CASE`   | `M3U_URL`, `SCRUB_HEADER`, `LIVE_TV`        |
| Files          | `snake_case`   | `classifier.go`, `config_test.go`           |
| Packages       | `lowercase`    | `config`, `entry`, `m3u`, `strm`            |

### Package Design

- All application packages live under `internal/` (not importable externally)
- Each package has a clear, single responsibility
- Public functions have doc comments
- No `init()` functions except for compiled regex patterns at package level
- Regex patterns are compiled once as package-level `var` using `regexp.MustCompile`

### Regex — RE2 Only

Go's `regexp` package uses RE2 syntax. **Lookaheads (`(?=`), lookbehinds (`(?<=`), and
negative lookaheads (`(?!`) are NOT supported.** Use `FindStringIndex` + substring
extraction instead. This was a key lesson from the Python-to-Go port.

### Error Handling

- Functions return `error` as the last return value
- Errors are wrapped with `fmt.Errorf("context: %w", err)` for stack context
- Non-fatal errors are logged with `slog.Error` or `slog.Warn` and processing continues
- Fatal errors are returned up to `main()` which calls `os.Exit(1)`
- The `defer/recover` pattern is used in `ClassifyAndClean` to catch panics from
  unexpected input

### Logging

- Uses `log/slog` (structured logging) throughout
- Log levels: DEBUG, INFO, WARN, ERROR (configurable via `LOG_LEVEL` env var)
- Logging is configured in `main.go` via `setupLogging()`
- Log key-value pairs for structured context: `slog.Info("msg", "key", value)`

### Configuration

- All config comes from environment variables (no config files, no flags)
- `config.Load()` returns a typed `*Config` struct — no raw string lookups elsewhere
- CSV parsing supports escaped commas (`\,`), outer-quote stripping
- `REPLACE_TERMS` uses `key=value` pair format: `"old1=new1,old2=new2"`
- Boolean env vars accept: `true`, `yes`, `1`, `t` (case-insensitive)

### Data Model

- `entry.Entry` is the central data structure — a struct with typed fields
- Entry classification produces one of: `TypeSeries`, `TypeTV`, `TypeMovie`,
  `TypeLiveTV`, `TypeUnsorted`
- No interfaces or generics are used; the codebase is straightforward procedural Go

### Testing

- Tests use the standard `testing` package (no testify or other frameworks)
- Test files are colocated with source: `classifier_test.go` next to `classifier.go`
- Tests are in the same package (white-box testing): `package entry`, not `package entry_test`
- Table-driven tests are preferred for multiple cases
- Use `t.TempDir()` for filesystem tests

### Dependencies

The project has **no external dependencies** — only Go stdlib. The `go.mod` file has
no `require` directives. Keep it that way unless there's a strong reason to add one.

### Docker

- Multi-stage build: `golang:1.23-alpine` (build) → `alpine:3.20` (runtime)
- Static binary with `CGO_ENABLED=0`
- Default `DATA_DIR=/data`; volume mount at `/data/VODS` for media output
- All env vars have defaults in the Dockerfile

### Processing Pipeline (main.go)

1. Load config from env vars
2. Create required directories
3. Download M3U files from URLs
4. Combine downloaded files into one
5. Parse combined M3U → classify entries
6. Write `.strm` files (series/TV/movies/unsorted)
7. Write live TV M3U playlist
8. Sync staging dirs to local (VODS) dirs
9. Report counts and errors
10. Cleanup intermediate files
11. Run integrations (Jellyfin refresh, Threadfin update)
