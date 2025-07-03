# Best Practices

This document lists recommended practices when running or modifying Goobla.

## Security

- **Disable profiling** unless explicitly required. Set `GOOBLA_PPROF=off` in production and run pprof on a separate port when needed.
- **Protect registry and pull/push endpoints** behind authentication and TLS. Set `GOOBLA_TLS_CERT` and `GOOBLA_TLS_KEY` to serve HTTPS and never expose these endpoints directly to the public Internet.
- **Validate configuration values** at startup and fail fast on invalid settings.
- **Avoid committing credentials**. Example keys in documentation are placeholders and should not be reused.

## Performance and Reliability

- Use the built‑in prompt cache when serving multiple users (`GOOBLA_MULTIUSER_CACHE=1`).
- Run integration tests and linters (`go vet`, `staticcheck`) before deploying changes.
- Monitor GPU and memory usage to size hardware appropriately.

## Logging

- Prefer structured logging through `logutil.NewLogger`.
- Include context such as request IDs when logging errors.

## Code Quality

- Write concurrency tests for packages that interact with the cache or scheduler.
- Document public functions and keep the `docs/` folder up to date with code changes.

Following these guidelines helps keep deployments secure and maintainable.
