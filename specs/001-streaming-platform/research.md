# Research: dotslashstream Core Platform

**Branch**: `001-streaming-platform` | **Date**: 2024-01-18  
**Feature**: dotslashstream Core Platform

## Research Topics

### 1. anacrolix/torrent Sequential Download

**Decision**: Use ` torrent.SetSequentialDownload()` API  
**Rationale**: Forces piece download in order (1, 2, 3...) instead of random. Essential for FFmpeg to consume data linearly.  
**Alternatives considered**:
- Manual piece priority manipulation — More complex, same result
- Waiting for full download — Defeats purpose of streaming

**Implementation**:
```go
torrent.SetSequentialDownload()
// Set high priority for first 5% of pieces
for i := 0; i < totalPieces/20; i++ {
    torrent.PiecePriority(i, 7) // Highest priority
}
```

### 2. FFmpeg HLS Piping

**Decision**: Pipe torrent data directly to FFmpeg stdin  
**Rationale**: Avoids writing to disk twice. FFmpeg reads sequentially from pipe, outputs HLS segments.  
**Alternatives considered**:
- Download first, then transcode — Adds latency, requires disk space
- Memory-mapped files — Complex, potential memory issues

**Implementation**:
```go
cmd := exec.Command("ffmpeg",
    "-i", "pipe:0",           // Read from stdin
    "-c:v", "libx264",
    "-crf", "28",
    "-preset", "ultrafast",
    "-f", "hls",
    "-hls_time", "4",
    "-hls_segment_filename", "/tmp/segments/%03d.ts",
    "/tmp/hls/master.m3u8",
)
stdin, _ := cmd.StdinPipe()
// Feed torrent data to stdin
```

### 3. asynq Task Patterns

**Decision**: Define 4 task types with progress reporting  
**Rationale**: Clear separation of concerns, easy monitoring via asynqmon.  
**Alternatives considered**:
- Single generic task type — Harder to monitor/retry
- Custom queue system — Reinventing the wheel

**Task Types**:
```go
const (
    TaskStream   = "stream:process"   // Start streaming
    TaskArchive  = "archive:process"  // Archive after download
    TaskCleanup  = "cleanup:segments" // Remove temp files
    TaskHealth   = "worker:heartbeat" // Worker liveness
)
```

**Progress Reporting**: Use Redis Pub/Sub separate from asynq for real-time WebSocket updates.

### 4. MinIO Presigned URLs

**Decision**: Generate presigned URLs for HLS segment access  
**Rationale**: Secure, time-limited access without exposing storage credentials.  
**Alternatives considered**:
- Proxy through API — Adds latency, single point of failure
- Public bucket — Security risk

**Implementation**:
```go
presignedURL, err := minioClient.PresignedGetObject(
    ctx, bucket, key, 
    time.Hour, // 1 hour expiry
    url.Values{},
)
```

### 5. Colly Scraping Patterns

**Decision**: Use Colly with configurable selectors per indexer  
**Rationale**: Mature library, handles rate limiting, easy to configure.  
**Alternatives considered**:
- Custom HTTP + BeautifulSoup-ported selectors — More work
- Goquery only — Less features than Colly

**Config Format**:
```json
{
  "name": "1337x",
  "type": "html",
  "search_path": "/search/{query}/1/",
  "selectors": {
    "result": "td.name",
    "title": "a:last-child",
    "url": "a:last-child@href",
    "seeds": "td.seeds",
    "size": "td.size"
  }
}
```

### 6. Docker Network Isolation

**Decision**: Three isolated networks (dmz, internal, storage)  
**Rationale**: Defense in depth. Worker/search cannot reach internet or each other.  
**Alternatives considered**:
- Single network with iptables — Complex, error-prone
- No isolation — Security risk

**Network Config**:
```yaml
networks:
  dmz:
    driver: bridge
  internal:
    driver: bridge
    internal: true  # No external access
  storage:
    driver: bridge
    internal: true

services:
  api:
    networks: [dmz, internal, storage]
  worker:
    networks: [internal, storage]  # No dmz = no internet
  search:
    networks: [internal]  # No storage, no internet
```

### 7. Go net/http Patterns

**Decision**: Use standard library with manual routing  
**Rationale**: No external dependencies, full control, good enough for this scale.  
**Alternatives considered**:
- Chi/Gin — Extra dependency, not needed
- Custom router — Reinventing

**Structure**:
```go
mux := http.NewServeMux()
mux.HandleFunc("POST /auth/register", handlers.Register)
mux.HandleFunc("POST /auth/login", handlers.Login)
mux.HandleFunc("GET /media/{id}", handlers.GetMedia)
// etc.

// Middleware chain
handler := logging(middleware(auth(mux)))
```

### 8. sqlc + PostgreSQL

**Decision**: Use sqlc for type-safe SQL queries  
**Rationale**: Compile-time query validation, generated Go code, no ORM overhead.  
**Alternatives considered**:
- GORM — Too much magic, harder to debug
- Raw SQL strings — No type safety
- pgx only — Still need query organization

**Workflow**:
1. Write SQL in `queries.sql`
2. Run `sqlc generate`
3. Use generated Go functions

## Summary of Decisions

| Topic | Decision | Rationale |
|-------|----------|-----------|
| Torrent | anacrolix/torrent + sequential | Native Go, streaming support |
| Transcode | FFmpeg pipe stdin | No double disk write |
| Queue | asynq + Redis | Built-in monitoring |
| Storage | MinIO presigned URLs | Secure, time-limited |
| Scraping | Colly + configurable selectors | Mature, flexible |
| Networks | 3 isolated Docker networks | Defense in depth |
| HTTP | net/http standard library | No dependencies |
| SQL | sqlc + PostgreSQL | Type-safe, generated code |
