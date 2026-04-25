# pricemonitor improvement plan

A staged plan to take this project from "working prototype" to "production-grade".
Each numbered task is sized to be a single focused commit. Tasks are ordered by
risk/impact: data-correctness and security first, then reliability, then quality
of life (tests, docs, build, CI), then refactors.

Tick boxes as you go. Each task lists concrete file/line targets and a short
verification step.

---

## Phase 1 — Correctness bugs (do these first)

- [ ] **1. Don't emit zero-valued samples on scrape error**
  - Files: `main.go:147-151`
  - Today: when `station.ScrapePrices()` returns an error, the worker still does
    `funnel <- sample` with a zero-valued `Sample{}`. The collector then upserts
    a junk station row with empty brand/address/geo and a real UUID.
  - Fix: in the worker, on error, log and `continue` instead of forwarding the
    sample. Optionally count failures via a metric/log field.
  - Verify: temporarily make one station fail; confirm `pricemonitor_stations`
    no longer gains an empty-string row.

- [ ] **2. Bound the Aral retry loops**
  - Files: `internal/stations/aral.go:29-48`, `:93-111`
  - Today: `goto retry` with a 5s sleep on every non-200, no max attempts, no
    context. A misconfigured station blocks a worker forever.
  - Fix: replace `goto` with a `for attempt := 0; attempt < N; attempt++` loop;
    add jittered exponential backoff; honor a `context.Context` (see task 14);
    return an error after the budget is spent.
  - Verify: point a station at an invalid URL; worker errors out within ~30s.

- [ ] **3. Same retry treatment for Shell**
  - Files: `internal/stations/shell.go` (request site, ~`:1444-1461`)
  - Audit Shell's request path for the same unbounded retry / unhandled error
    shapes. Apply the same bounded retry + backoff helper from task 2.

- [ ] **4. Fix Shell response body leak on read error**
  - Files: `internal/stations/shell.go:1455-1461`
  - Today: `Body.Close()` only runs after `ReadAll` succeeds.
  - Fix: `defer resp.Body.Close()` immediately after the err check on the
    `Do(req)` call. Same pattern in `aral.go:50-52` and `:113-115` for
    consistency.

- [ ] **5. Don't poison batches when `UpsertStation` fails**
  - Files: `main.go:186-205`, collector loop
  - Today: an upsert error is logged but the loop keeps building a batch where
    `station_id` is the zero UUID, then the FK insert fails for the whole
    batch.
  - Fix: on upsert error, drop just that station's samples from the batch; keep
    going. Log with station identifier.

- [ ] **6. Rename or remove the misleading `Sample.ID`**
  - Files: `internal/stations/aral.go:157`, `internal/stations/shell.go:1488`,
    `internal/model/queries.sql`, migration for `pricemonitor_samples`.
  - Today: a fresh UUID is generated per scrape and reused for every sample
    row; the column is `NOT NULL` but not a PK — the field is meaningless.
  - Fix: either (a) drop the column entirely (new migration + sqlc regen), or
    (b) rename to `scrape_id` to make the per-scrape grouping explicit.
  - Verify: `sqlc generate` clean; migration up/down works on a scratch DB.

---

## Phase 2 — Security & operational

- [ ] **7. Remove `InsecureSkipVerify: true` for Shell**
  - Files: `internal/stations/shell.go:1444-1447`
  - Today: TLS validation disabled for Shell scraping with no justification.
  - Fix: drop the override; if Shell genuinely fails cert validation, capture
    the failure (`openssl s_client`) and either trust the right intermediate or
    document the upstream issue in a code comment with a dated TODO. Do not
    ship `InsecureSkipVerify` to production.

- [ ] **8. Reuse a single `http.Client` per brand**
  - Files: `internal/stations/aral.go`, `shell.go` (per-request `Transport` /
    `Client` construction).
  - Fix: build one `*http.Client` at `NewStation` time (or a package-level
    `var`) with a sane `Timeout` and tuned `Transport`. Pass it into the
    request methods. Restores connection pooling and TLS session reuse.

- [ ] **9. Build the DSN with `pgx.ParseConfig`, not `fmt.Sprintf`**
  - Files: `main.go:75-80, 101-108`
  - Today: a password containing spaces, `'`, or `\` breaks parsing or allows
    libpq keyword injection (e.g. `password=foo sslmode=disable`).
  - Fix: `cfg, err := pgx.ParseConfig("")`; assign `cfg.Host`, `cfg.User`,
    `cfg.Password`, `cfg.Database`, `cfg.Port` from struct fields; use
    `pgx.ConnectConfig(ctx, cfg)`. Same for the `database/sql` handle used by
    goose: `stdlib.RegisterConnConfig` + `sql.Open("pgx", connStr)`.
  - Add an explicit `sslmode` (env-driven, default `prefer`).

- [ ] **10. Stop hardcoding `dbname=postgres`**
  - Files: `main.go:75-80`, `Config` struct.
  - Fix: add a `Database` field to `Config.Postgres` (default `postgres` for
    compatibility) and wire it into the parsed config from task 9.

- [ ] **11. Add graceful shutdown**
  - Files: `main.go` (top-level), worker/collector loops.
  - Fix: `ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)`.
    Pass `ctx` to workers, the collector, the HTTP requests, and `pgx`. On
    cancel: stop accepting new work, drain the funnel, flush the in-flight
    batch, then exit. Log a clean-shutdown line.
  - Verify: `Ctrl-C` mid-batch flushes pending samples and exits within a few
    seconds.

---

## Phase 3 — Concurrency refactor

- [ ] **12. Long-lived worker pool driven by a ticker**
  - Files: `main.go:130-174`
  - Today: workers, channels, and the WaitGroup are recreated every minute.
  - Fix: spawn N workers once at startup. Drive scrape rounds with a
    `time.NewTicker(interval)`. On each tick, push the station list into a
    buffered work channel. Workers loop on `select { case s := <-work: ...
    case <-ctx.Done(): return }`.

- [ ] **13. Buffer channels and rethink batching**
  - Files: `main.go:126-233`
  - Fix: make `funnel` buffered (e.g. `len(stations)`); remove the ad-hoc
    `done` chan in favor of `ctx.Done()` and `close(work)`/`close(funnel)` at
    shutdown. The collector becomes a single goroutine batching by
    `(maxSize || maxAge)` and flushing on either trigger or `ctx.Done()`.
  - Document the batching state machine in a short doc-comment above the
    collector.

- [ ] **14. Plumb `context.Context` everywhere**
  - Files: `internal/stations/stations.go` (interface),
    `internal/stations/aral.go`, `shell.go`, `main.go` callsites.
  - Fix: change `Station.ScrapePrices()` to
    `ScrapePrices(ctx context.Context) (Sample, error)`. Use `http.NewRequestWithContext`
    in both scrapers. Use `pgx`'s context-aware methods in the collector.

---

## Phase 4 — Tests (highest leverage for reliability)

- [ ] **15. Add a tiny test scaffold**
  - New: a `Makefile` (or `Taskfile.yml`) with `make test`, `make lint`,
    `make build`, `make sqlc`. Add `golangci-lint` config (`.golangci.yml`)
    with sane defaults including the `ireturn` linter already implied by the
    existing `//nolint` directive.

- [ ] **16. Aral parser tests with captured fixtures**
  - New: `internal/stations/testdata/aral/<station>.html` and
    `…/prices.json` captured from a real response (sanitized).
  - New: `internal/stations/aral_test.go` — table-driven tests that load
    fixtures from disk, run the parser entry-point, and assert on
    `Sample`/fuel rows. Spin up an `httptest.Server` to serve the fixtures so
    the full request path is covered.
  - Verify: `go test ./internal/stations/ -run Aral` is green and runs offline.

- [ ] **17. Shell parser tests with captured fixtures**
  - Same approach for `shell.go`. Crucial because Shell's payload is a giant
    embedded React-props blob — the test guards against silent layout drift.

- [ ] **18. Collector / batching unit tests**
  - New: `collector_test.go` (move collector to its own package first — see
    task 23). Cover: max-size flush, max-age flush, error-on-upsert drops only
    that station, context cancel flushes pending batch.

- [ ] **19. Migration smoke test**
  - New: a test that boots a Postgres+TimescaleDB container (testcontainers-go
    or a `t.Skip` if `PRICEMONITOR_TEST_DB` is unset) and runs migrations
    up/down. Catches future migration breakage early.

---

## Phase 5 — Build, deploy, CI

- [ ] **20. Replace the Dockerfile (or commit fully to `ko`)**
  - Files: `Dockerfile`, `.envrc`.
  - Decision first: the `.envrc` configures `ko` (`KO_DOCKER_REPO`,
    `KO_DEFAULT_BASE_IMAGE=alpine`). If `ko` is the real path, **delete the
    Dockerfile** and add a `make image` target invoking `ko build`.
  - If keeping Docker: rewrite as a multi-stage build, pin `golang:1.23-alpine`
    + `gcr.io/distroless/static-debian12` (or `alpine`), drop the Chrome
    install (dead weight from the removed browserless flow), `COPY` not `ADD`,
    add a `.dockerignore`, run as a non-root `USER`.
  - Verify: final image < 50 MB, runs as non-root, starts and connects to a
    test DB.

- [ ] **21. Add a `.dockerignore`**
  - New: `.dockerignore` excluding `.git`, `*.md`, `testdata/`, `PLAN.md`,
    local env files, etc.

- [ ] **22. GitHub Actions CI**
  - New: `.github/workflows/ci.yml` running on push + PR:
    - `actions/setup-go@v5` with `go-version-file: go.mod`
    - `go vet ./...`
    - `golangci-lint run`
    - `go test ./... -race -count=1`
    - `ko build` (or `docker build`) on `main` only.
  - Optional: a `release` workflow that pushes the image to GHCR on tag.

---

## Phase 6 — Refactor & code organization

- [ ] **23. Split `main.go` into packages**
  - New: `internal/config`, `internal/worker`, `internal/collector`,
    `internal/db` (DSN building, pgx connect, goose runner). `main.go` becomes
    ~50 lines of wiring.

- [ ] **24. Move the generated Shell React-props struct out of `shell.go`**
  - Files: `internal/stations/shell.go` (currently 1503 lines, ~98% generated
    struct).
  - Fix: move the struct to `internal/stations/shell_props.go` with a header
    comment noting it's hand-derived from the upstream React props blob.
    Leaves `shell.go` reviewable.

- [ ] **25. Move `BrandAral`/`BrandShell` constants to `stations.go`**
  - Files: `internal/stations/stations.go`, `aral.go:17`, `shell.go:17`.
  - Trivial cleanup: keep the `Brand` type and its values together.

- [ ] **26. Replace `[]byte → string → strings.NewReader` round-trips**
  - Files: `internal/stations/aral.go:58`, `shell.go:1463`.
  - Fix: `htmlquery.Parse(bytes.NewReader(buf))`.

- [ ] **27. Fix non-idiomatic error strings**
  - Files: `internal/stations/stations.go:62` (capitalized "Unknown brand"),
    `internal/stations/shell.go:32, 38` (empty `errors.New("")`).
  - Fix: lowercase, no trailing punctuation; give the empty errors real
    messages.

- [ ] **28. Drop unused indirect deps**
  - Files: `go.mod`, `go.sum`.
  - Run `go mod tidy` after the refactors. Confirm `go-retry` and
    `go.uber.org/multierr` are gone (or used intentionally).

---

## Phase 7 — Documentation

- [ ] **29. Write a README**
  - New: `README.md` covering: what it does, supported brands, the
    `PRICEMONITOR_STATIONS` format with examples, env vars, how to run
    locally (`docker compose up` for Postgres+Timescale), how to run tests,
    how the data model is laid out, and a note on the TimescaleDB single-DB
    constraint.

- [ ] **30. Add a LICENSE**
  - Pick a license (MIT/Apache-2.0/AGPL — your call) and add `LICENSE`.

- [ ] **31. Godoc on exported symbols**
  - Files: `internal/stations/stations.go` (`Station`, `Sample`, `Brand`,
    `NewStation`, `BrandAral`, `BrandShell`),
    `internal/model/migrations/migrations.go` (the embed FS),
    package-level doc comments for `package stations`, `package model`,
    `package migrations`.

- [ ] **32. Document the data model**
  - New: `docs/data-model.md` (or a section in README) describing the
    `pricemonitor_stations`, `pricemonitor_samples` hypertable, and the
    weekly/daily continuous aggregates, with an example query for "current
    price per station".

- [ ] **33. Fix the migration filename typo**
  - Files: `internal/model/migrations/20241022211130_create_shema.sql`.
  - Note: cannot rename in place without breaking goose's history. Either
    leave it with a doc comment, or add a follow-up migration that does
    nothing but renames future-facing references in code/docs.

---

## Phase 8 — Observability (nice to have)

- [ ] **34. Prometheus metrics endpoint**
  - Add `prometheus/client_golang`. Expose `/metrics` on a small HTTP server.
  - Counters: `scrape_attempts_total{brand,station,result}`,
    `samples_inserted_total`, `batches_flushed_total{reason=size|age|shutdown}`.
  - Gauges: `last_successful_scrape_timestamp_seconds{brand,station}`.
  - Histograms: `scrape_duration_seconds{brand}`, `batch_flush_duration_seconds`.

- [ ] **35. Health endpoint**
  - `/healthz` (process up) and `/readyz` (DB pingable, last successful scrape
    within 2x interval). Wire into the Docker/ko `HEALTHCHECK`.

---

## Suggested commit order

Phase 1 → 2 → 3 in that order is the safest path: bugs and security first,
then concurrency, because tests in Phase 4 are easier to write against the
context-plumbed, single-pool design from Phase 3. Phases 5–8 can be done in
parallel with each other once Phase 4 lands.
