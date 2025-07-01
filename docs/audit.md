# Code Audit Overview

This document outlines observations and improvement suggestions based on a quick audit of the Goobla code base.

## Observed Issues

### Profiling Endpoints Enabled by Default

The server exposes Go's `pprof` handlers by default when `GOOBLA_PPROF` is unset or set to `on`. This happens in the server startup logic:

```go
pprofAddr := strings.ToLower(envconfig.PprofAddr())
var srvr *http.Server
switch pprofAddr {
case "", "on":
    // Use DefaultServeMux so we get net/http/pprof handlers on the main server.
    http.Handle("/", h)
    srvr = &http.Server{Handler: nil}
...
```

_Source: `server/routes.go` lines 1276‑1308_

The configuration description for `GOOBLA_PPROF` states:

```go
// PprofAddr configures the pprof server address. Set to "off" to disable
// pprof or specify a custom address (e.g. 127.0.0.1:6060).
PprofAddr = String("GOOBLA_PPROF")
```

_Source: `envconfig/config.go` lines 189‑191_

Documentation reiterates that pprof is enabled by default:

```
## Profiling
By default the server exposes Go's pprof handlers on the main port. Set the
environment variable `GOOBLA_PPROF` to `off` to disable these endpoints or
specify a host and port such as `127.0.0.1:6060` to run pprof on a separate
port.
```

_Source: `docs/development.md` lines 160‑166_

Leaving profiling endpoints enabled on production servers can leak internal
information. A safer default is to disable pprof unless explicitly enabled.

### Cache Concurrency Concerns

`DiskCache` is documented as not preventing duplicated effort when used
concurrently:

```go
// The cache is safe for concurrent use.
// ...
// The cache is not safe for concurrent use. It guards concurrent writes, but
// does not prevent duplicated effort. Because blobs are immutable, duplicate
// writes should result in the same file being written to disk.
type DiskCache struct {
    // Dir specifies the top-level directory where blobs and manifest
    // pointers are stored.
    dir string
    now func() time.Time
```

_Source: `server/internal/cache/blob/cache.go` lines 40‑56_

Race conditions can occur when multiple processes attempt to write or link the
same blob simultaneously. Implementing stronger locking or deduplication at a
higher layer would improve robustness.

### Example API Key in Documentation

The OpenAI client example uses a placeholder API key:

```
const openai = new OpenAI({
  baseURL: 'http://localhost:11434/v1/',

  // required but ignored
  apiKey: 'goobla',
})
```

_Source: `docs/openai.md` lines 100‑109_

Ensure that real credentials are never committed to the repository and clarify
that this key is only a placeholder.

## Recommended Improvements

1. **Disable pprof by default** – change the default branch of the server to
   start without pprof handlers unless `GOOBLA_PPROF` is explicitly set.
2. **Add authentication/authorization** for sensitive endpoints (pull, push,
   profiling) and document how to enable TLS in production deployments.
3. **Harden DiskCache** – enforce write locks or use a database/registry to
   avoid duplicate writes when multiple workers import data concurrently.
4. **Improve logging** – adopt structured logging across packages and ensure
   errors always propagate relevant context.
5. **Expand testing** – add concurrency tests for the cache and integration
   tests covering API routes. Continuous linting (e.g., `go vet`, `staticcheck`)
   should be part of CI.
6. **Security documentation** – extend `SECURITY.md` with more detailed threat
   modeling and steps for patch management.
7. **Code cleanup** – address `TODO` comments, document public functions, and
   remove unused code paths where possible.
8. **Configuration validation** – fail fast on invalid environment variable
   values and provide clear error messages at startup.

Implementing these changes will make the application more secure, easier to
maintain, and more robust when deployed at scale.
