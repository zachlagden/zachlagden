# Profile views counter

A self-hosted SVG badge with one persistent SQLite counter. GitHub image-proxy requests increment it. Other requests read the current value.

## Behaviour

- `GET /badge.svg` increments the count only when `User-Agent` starts with `github-camo`.
- Other `GET` requests to the badge and all `HEAD` requests leave the count unchanged.
- Responses allow five minutes of caching. Cached views do not reach the counter.
- The user-agent check matches GitHub's proxy behaviour. It does not prevent someone from spoofing that header.
- The service stores no visitor identities, IP addresses or request history.
- The request limiter permits a burst of 20 requests and then two requests per second.
- No public health, admin, reset, docs or OpenAPI route exists. The API description is in `openapi.yaml`.
- Database errors produce generic HTTP responses. The process writes structured JSON errors to stdout.

This counts GitHub image-proxy fetches, not unique people or every profile visit.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_PATH` | `/data/views.sqlite3` | Persistent SQLite database path. |
| `INITIAL_COUNT` | `0` | Nonnegative starting count, used only when the counter row does not exist. |

An existing counter never resets when `INITIAL_COUNT` changes. A corrupt database fails instead of being replaced. Losing the entire data volume requires restoring a backup.

The HTTP server listens on port 8080. Production traffic must pass through an HTTPS reverse proxy.

## Checks

Run these commands in `counter/`:

```sh
go test -race -count=1 ./...
go vet ./...
```

Build from the repository root:

```sh
docker build -f counter/Dockerfile -t profile-views .
```

The container runs as user 65532 and has no shell. Its built-in healthcheck makes a local `HEAD` request and does not increment the count.

## Backup and restore

Create a consistent snapshot using the running container:

```sh
docker exec profile-views /profile-views backup /data/backups/snapshot.sqlite3
```

The backup command refuses to overwrite an existing file. It uses SQLite's `VACUUM INTO`, so it includes committed changes from the WAL without copying a live database file.

To restore, stop the application, preserve the current database and its WAL files outside the data directory, then put a validated snapshot at `DATABASE_PATH`. Set its ownership to 65532:65532 before restarting. The service reads the restored count and ignores `INITIAL_COUNT`.

## Production

This profile uses `https://github-readme-views.zachlagden.uk/badge.svg`, with an initial count of 5,866.

The Coolify application builds `/counter/Dockerfile` from this repository, exposes port 8080 to Traefik, and mounts a persistent volume at `/data`. Cloudflare proxies the HTTPS hostname. Configure Coolify to use the Dockerfile healthcheck rather than generating a curl-based one, because the image has no curl or shell.
