# Profile views counter

A Go service that returns an SVG badge at `/badge.svg` and stores its count in SQLite.

Only `GET` requests whose `User-Agent` starts with `github-camo` increment the counter. Other `GET` requests and all `HEAD` requests read its current value. Responses use `Cache-Control: no-cache` so caches must revalidate before reusing the badge.

This counts GitHub image-proxy fetches, not unique visitors. The user-agent check is not tamper-proof. The service stores no visitor identities or request history.

## Configuration

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_PATH` | `/data/views.sqlite3` | Persistent SQLite database path. |
| `INITIAL_COUNT` | `0` | Starting count, used only when the counter row does not exist. |

Existing counts survive restarts and ignore changes to `INITIAL_COUNT`. A corrupt database fails instead of being replaced.

The server listens on port 8080. Only `/badge.svg` is public. Other routes return 404, including health, admin, reset and docs routes. The API description is in `openapi.yaml`.

## Checks

Run in `counter/`:

```sh
go test -race -count=1 ./...
go vet ./...
```

Build from the repository root:

```sh
docker build -f counter/Dockerfile -t profile-views .
```

## Deployment

The badge is at `https://github-readme-views.zachlagden.uk/badge.svg`, with an initial count of 5,866.

Coolify app `78eovlfehtijc1zl9rhm7myl` builds `/counter/Dockerfile` from this repository. Its container is `github-readme-views`, with volume `78eovlfehtijc1zl9rhm7myl-github-readme-views-data` mounted at `/data`. Traefik handles HTTPS and Cloudflare proxies the hostname.
