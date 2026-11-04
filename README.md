# Logka

Self-hosted Docker log viewer. One binary (or container) that lists containers and streams live logs in a brand-first UI.

## Quick start

### Docker (recommended)

Mount the Docker socket so Logka can talk to the local Engine:

```bash
docker run --rm -p 8080:8080 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  ghcr.io/debelistic/logka
```

Open http://localhost:8080

### From source

Requires Go 1.22+ and Node 20+.

```bash
# build UI into internal/server/dist (embedded by the Go binary)
cd web && npm install && npm run build && cd ..

go run ./cmd/logka
# listens on :8080 — override with -addr or LOGKA_ADDR
```

### Multi-stage image build

```bash
docker build -t logka .
docker run --rm -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock logka
```

## Features (v1 scaffold)

- List Docker containers (running / all)
- Live log streaming over SSE (`/api/containers/{id}/logs`)
- Container filter + log search foundations
- Single-binary deploy with embedded React UI

## API

| Method | Path | Description |
|--------|------|-------------|
| `GET` | `/api/health` | Service + Docker ping |
| `GET` | `/api/containers` | Container list |
| `GET` | `/api/containers/{id}/logs?tail=200&follow=true` | SSE log stream |

## Layout

```
cmd/logka/           # entrypoint
internal/docker/     # Docker Engine client
internal/api/        # HTTP + SSE
internal/server/     # embeds web build (dist/)
web/                 # React + Vite UI
Dockerfile
```

## Development

```bash
# terminal 1 — API
go run ./cmd/logka

# terminal 2 — Vite (proxies /api → :8080)
cd web && npm run dev
```

## License

MIT
