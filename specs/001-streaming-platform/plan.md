# Implementation Plan: dotslashstream Core Platform

**Branch**: `001-streaming-platform` | **Date**: 2024-01-18 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/specs/001-streaming-platform/spec.md`

## Summary

dotslashstream is a self-hosted media streaming service that enables streaming while downloading torrent content. The system uses HLS transcoding for adaptive bitrate playback, supports movies and TV shows with season/episode hierarchy, and provides both search-based and manual magnet link input. Architecture follows network isolation principles with API Gateway as the sole entry point.

## Technical Context

**Language/Version**: Go 1.22+  
**Primary Dependencies**: net/http, anacrolix/torrent, minio-go/v7, asynq, bun, Colly, caarlos0/env  
**Storage**: PostgreSQL (metadata), Redis (queue/cache), MinIO (media files)  
**Testing**: go test, testify  
**Target Platform**: Linux server (Docker)  
**Project Type**: microservices (4 services)  
**Performance Goals**: <10s stream start, 5 concurrent streams, <3s search  
**Constraints**: 3 concurrent downloads per user, network isolation  
**Scale/Scope**: Personal use, 5-10 users, single-server deployment

## Constitution Check

*GATE: Constitution not yet configured. Skipping gate checks.*

## Project Structure

### Documentation (this feature)

```text
specs/001-streaming-platform/
├── plan.md              # This file
├── research.md          # Phase 0 output
├── data-model.md        # Phase 1 output
├── quickstart.md        # Phase 1 output
├── contracts/           # Phase 1 output
└── tasks.md             # Phase 2 output (not created by this command)
```

### Source Code (repository root)

```text
.
├── apps/
│   └── api/                          # API Gateway service
│       ├── cmd/
│       │   └── api/
│       │       └── main.go
│       ├── internal/
│       │   ├── app/
│       │   │   ├── app.go            # wires everything, starts server
│       │   │   ├── config.go         # env parsing
│       │   │   └── routes.go         # registers handlers on mux
│       │   ├── auth/
│       │   │   ├── handler.go        # HTTP handlers
│       │   │   ├── service.go        # business logic
│       │   │   ├── store.go          # UserStore interface
│       │   │   ├── jwt.go            # token issuer
│       │   │   └── types.go          # User, Claims
│       │   ├── media/
│       │   │   ├── handler.go
│       │   │   ├── service.go
│       │   │   ├── store.go          # MediaStore interface
│       │   │   └── types.go
│       │   ├── playlist/
│       │   │   ├── handler.go
│       │   │   ├── service.go
│       │   │   ├── store.go          # PlaylistStore interface
│       │   │   └── types.go
│       │   ├── stream/
│       │   │   ├── handler.go
│       │   │   ├── service.go
│       │   │   ├── store.go          # StreamStore interface
│       │   │   └── types.go
│       │   └── platform/
│       │       ├── database.go       # DatabaseClient interface
│       │       ├── queue.go          # QueueClient interface
│       │       ├── bucket.go         # BucketClient interface
│       │       ├── postgres/
│       │       │   ├── driver.go     # implements DatabaseClient
│       │       │   ├── auth.go       # implements auth.UserStore
│       │       │   ├── media.go      # implements media.MediaStore
│       │       │   └── playlist.go   # implements playlist.Store
│       │       ├── sqlite/
│       │       │   ├── driver.go     # implements DatabaseClient
│       │       │   ├── auth.go       # implements auth.UserStore
│       │       │   ├── media.go      # implements media.MediaStore
│       │       │   └── playlist.go   # implements playlist.Store
│       │       ├── redis/
│       │       │   └── driver.go     # implements QueueClient
│       │       └── minio/
│       │           └── driver.go     # implements BucketClient
│       └── go.mod
│
├── worker/                           # Torrent Worker service
│   ├── cmd/
│   │   └── worker/
│   │       └── main.go
│   ├── internal/
│   │   ├── torrent/
│   │   │   ├── client.go
│   │   │   ├── download.go
│   │   │   └── sequential.go
│   │   ├── transcode/
│   │   │   ├── ffmpeg.go
│   │   │   ├── profiles.go
│   │   │   └── hls.go
│   │   ├── storage/
│   │   │   └── minio.go
│   │   ├── queue/
│   │   │   └── asynq.go
│   │   └── models/
│   │       ├── job.go
│   │       └── progress.go
│   └── go.mod
│
├── search/                           # Search Gateway service
│   ├── cmd/
│   │   └── gateway/
│   │       └── main.go
│   ├── internal/
│   │   ├── indexers/
│   │   │   ├── manager.go
│   │   │   ├── html.go
│   │   │   └── api.go
│   │   ├── search/
│   │   │   ├── engine.go
│   │   │   └── normalize.go
│   │   ├── cache/
│   │   │   └── redis.go
│   │   └── models/
│   │       ├── indexer.go
│   │       └── result.go
│   ├── configs/
│   │   └── indexers.json
│   └── go.mod
│
├── web/                              # Vue.js frontend
│   ├── src/
│   │   ├── components/
│   │   │   ├── player/
│   │   │   │   ├── VideoPlayer.vue
│   │   │   │   └── Controls.vue
│   │   │   ├── media/
│   │   │   │   ├── MediaCard.vue
│   │   │   │   └── MediaGrid.vue
│   │   │   ├── playlist/
│   │   │   │   ├── PlaylistList.vue
│   │   │   │   └── PlaylistEditor.vue
│   │   │   └── ui/
│   │   │       ├── ProgressBar.vue
│   │   │       └── FormatSelector.vue
│   │   ├── views/
│   │   │   ├── Home.vue
│   │   │   ├── Login.vue
│   │   │   ├── Library.vue
│   │   │   ├── Media.vue
│   │   │   └── Settings.vue
│   │   ├── stores/
│   │   │   ├── auth.js
│   │   │   ├── media.js
│   │   │   └── player.js
│   │   ├── services/
│   │   │   ├── api.js
│   │   │   └── ws.js
│   │   └── App.vue
│   ├── index.html
│   └── package.json
│
├── docker-compose.yml               # Service orchestration
├── .env.example                      # Environment template
└── docs/                             # Documentation
```

**Structure Decision**: Multi-service architecture with 4 isolated services (api, worker, search, web). API uses vertical feature slices (auth, media, playlist, stream) with shared platform adapters for database, queue, and bucket. Feature packages define store interfaces; platform/ provides driver implementations. Swap Postgres ↔ SQLite by changing one line in app.go.

## Research Topics

### Phase 0: Research

1. **anacrolix/torrent sequential download** — How to force sequential piece delivery for streaming
2. **FFmpeg HLS piping** — Real-time transcoding from torrent stream to HLS segments
3. **asynq task patterns** — Job types, progress reporting, retry logic
4. **MinIO presigned URLs** — Secure segment access without exposing storage
5. **Colly scraping patterns** — HTML parsing for indexer support
6. **Docker network isolation** — Internal-only networks for worker/search
7. **Go net/http patterns** — Middleware, routing, WebSocket handling
8. **sqlc + PostgreSQL** — Type-safe SQL queries for Go

## Complexity Tracking

> *No constitution violations to justify.*
