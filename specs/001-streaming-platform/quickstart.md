# Quickstart: dotslashstream

## Prerequisites

- Docker 20.10+
- Docker Compose v2
- 16GB+ RAM recommended
- 100GB+ storage

## Setup

### 1. Clone and Configure

```bash
git clone https://github.com/artumont/dotslashstream.git
cd dotslashstream

cp .env.example .env
# Edit .env with your settings
```

### 2. Start Services

```bash
docker compose up -d
```

### 3. Initialize Database

```bash
docker compose exec api ./migrate up
```

### 4. Access

- **Web UI**: http://localhost:3000
- **API**: http://localhost:8000
- **MinIO Console**: http://localhost:9001

## First Steps

### 1. Create Account

```bash
curl -X POST http://localhost:8000/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com", "password": "securepass"}'
```

### 2. Login

```bash
curl -X POST http://localhost:8000/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email": "you@example.com", "password": "securepass"}'
```

### 3. Add Indexer

Edit `search/configs/indexers.json`:

```json
{
  "indexers": [
    {
      "name": "1337x",
      "type": "html",
      "url": "https://1337x.to",
      "search_path": "/search/{query}/1/",
      "selectors": {
        "result": "td.name",
        "title": "a:last-child",
        "url": "a:last-child@href",
        "seeds": "td.seeds",
        "size": "td.size"
      }
    }
  ]
}
```

### 4. Search and Stream

```bash
# Search
curl -X POST http://localhost:8000/search \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"query": "movie name"}'

# Stream via magnet
curl -X POST http://localhost:8000/stream/magnet \
  -H "Authorization: Bearer {token}" \
  -H "Content-Type: application/json" \
  -d '{"magnet_url": "magnet:?xt=..."}'
```

## Architecture

```
┌─────────────┐     ┌─────────────┐
│   Web App   │────▶│  API Gateway│
│   (Vue)     │     │    (Go)     │
└─────────────┘     └──────┬──────┘
                           │
                    ┌──────┴──────┐
                    │   Asynq     │
                    │   (Redis)   │
                    └──────┬──────┘
                           │
              ┌────────────┴────────────┐
              │                         │
              ▼                         ▼
       ┌─────────────┐          ┌─────────────┐
       │   Worker    │          │   Search    │
       │    (Go)     │          │    (Go)     │
       └─────────────┘          └─────────────┘
```

## Development

### API Gateway

```bash
cd api
go run ./cmd/server
```

### Worker

```bash
cd worker
go run ./cmd/worker
```

### Search Gateway

```bash
cd search
go run ./cmd/gateway
```

### Web Frontend

```bash
cd web
npm install
npm run dev
```

## Troubleshooting

### Port Conflicts

```bash
lsof -i :8000
lsof -i :3000
lsof -i :9000
```

### Logs

```bash
docker compose logs -f api
docker compose logs -f worker
docker compose logs -f search
```

### Reset

```bash
docker compose down -v
docker compose up -d
```
