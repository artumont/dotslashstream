# Services

## Overview

Four containerized services. API Gateway is the only externally accessible service.

```mermaid
graph TB
    subgraph "Exposed"
        API[API Gateway<br/>Go]
        Web[Web App<br/>Vue + Vite]
    end
    
    subgraph "Isolated"
        Worker[Torrent Worker<br/>Go]
        Search[Search Gateway<br/>Go]
    end
    
    subgraph "Storage"
        DB[(PostgreSQL)]
        Cache[(Redis)]
        S3[(MinIO)]
    end
    
    Web -->|HTTPS| API
    API -->|Internal| Worker
    API -->|Internal| Search
    API --> DB
    API --> Cache
    Worker --> S3
    Worker --> Cache
    
    style Worker fill:#ff9999
    style Search fill:#ff9999
```

---

## API Gateway

**Purpose:** Single entry point for all client communication.

**Tech:** Go + net/http + sqlc + go-redis

### Responsibilities

- JWT authentication and sessions
- User profile CRUD
- Playlist management
- Watch history tracking
- Job queue coordination
- WebSocket connections for real-time updates
- Rate limiting (3 concurrent downloads per user)
- Request validation
- Structured logging and basic metrics

### Network Access

```mermaid
graph TB
    subgraph "API Gateway"
        API[API Gateway]
    end
    
    subgraph "Allowed Connections"
        A1[Client: Accept HTTP/HTTPS]
        A2[PostgreSQL: Read/Write]
        A3[Redis: Read/Write]
        A4[Search Gateway: Forward queries]
    end
    
    subgraph "Blocked"
        B1[Internet: Outbound only for specific tasks]
    end
    
    API --> A1
    API --> A2
    API --> A3
    API --> A4
    
    style B1 fill:#ff6666
```

### Endpoints

```mermaid
graph TB
    subgraph "API Endpoints"
        Auth[/auth/*<br/>Register, Login, Refresh/]
        Users[/users/*<br/>Profile, Preferences/]
        Playlists[/playlists/*<br/>CRUD, Items/]
        Media[/media/*<br/>Details, Formats, Stream/]
        Library[/library<br/>Saved content/]
        Search[/search<br/>Query indexers/]
        HLS[/hls/*<br/>Serve segments/]
    end
    
    Auth --> Users
    Users --> Playlists
    Media --> Library
    Media --> HLS
```

### Configuration

```yaml
DATABASE_URL: postgresql://user:pass@db:5432/torrentstream
REDIS_URL: redis://redis:6379
JWT_SECRET: your-secret-key
SEARCH_GATEWAY_URL: http://search:8000
```

### Go Project Structure

```
api/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── auth/
│   │   ├── jwt.go
│   │   └── middleware.go
│   ├── handlers/
│   │   ├── auth.go
│   │   ├── users.go
│   │   ├── playlists.go
│   │   ├── media.go
│   │   └── search.go
│   ├── models/
│   │   ├── user.go
│   │   ├── playlist.go
│   │   └── media.go
│   ├── repository/
│   │   ├── user.go
│   │   ├── playlist.go
│   │   └── media.go
│   ├── service/
│   │   ├── auth.go
│   │   ├── user.go
│   │   ├── playlist.go
│   │   └── media.go
│   └── websocket/
│       └── hub.go
├── migrations/
├── go.mod
└── go.sum
```

---

## Torrent Worker

**Purpose:** Download torrents and transcode to HLS streams.

**Tech:** Go + anacrolix/torrent + ffmpeg bindings

### Network Access

```mermaid
graph TB
    subgraph "Torrent Worker"
        Worker[Torrent Worker]
    end
    
    subgraph "Allowed Connections"
        A1[Redis: Read jobs, write progress]
        A2[MinIO: Upload/download segments]
    end
    
    subgraph "Blocked"
        B1[Internet: Direct access]
        B2[PostgreSQL: No access]
        B3[Client traffic: No access]
    end
    
    Worker --> A1
    Worker --> A2
    
    style B1 fill:#ff6666
    style B2 fill:#ff6666
    style B3 fill:#ff6666
```

### Responsibilities

- Torrent lifecycle management
- Sequential piece delivery for streaming
- FFmpeg transcoding to HLS
- Object storage upload/download
- Progress reporting via Redis
- Health check heartbeats
- Structured logging

### Job Types

```mermaid
stateDiagram-v2
    [*] --> Idle
    
    Idle --> Streaming: stream job
    Idle --> Archiving: archive job
    Idle --> Cleanup: cleanup job
    
    Streaming --> Transcoding: buffer ready
    Transcoding --> Streaming: segments created
    Streaming --> Complete: stream done
    Streaming --> Failed: error
    
    Archiving --> Transcoding: re-encode
    Transcoding --> Archived: upload done
    
    Cleanup --> Idle: done
    
    Complete --> Idle
    Archived --> Idle
    Failed --> Idle
```

### Transcoding Profiles

| Profile | Preset | CRF | Use Case |
|---------|--------|-----|----------|
| stream-only | ultrafast | 28 | Immediate playback |
| archive-1080p | slow | 23 | Permanent storage |
| archive-720p | slow | 26 | Balanced quality |
| archive-480p | slow | 30 | Space efficient |

### Configuration

```yaml
REDIS_URL: redis://redis:6379
MINIO_ENDPOINT: minio:9000
MINIO_BUCKET: media
MAX_ACTIVE_DOWNLOADS: 5
SEED_RATIO: 1.5
SEED_TIME: 3600
```

### Go Project Structure

```
worker/
├── cmd/
│   └── worker/
│       └── main.go
├── internal/
│   ├── torrent/
│   │   ├── client.go
│   │   ├── download.go
│   │   └── sequential.go
│   ├── transcode/
│   │   ├── ffmpeg.go
│   │   ├── profiles.go
│   │   └── hls.go
│   ├── storage/
│   │   └── minio.go
│   ├── queue/
│   │   └── asynq.go
│   └── models/
│       ├── job.go
│       └── progress.go
├── go.mod
└── go.sum
```

---

## Search Gateway

**Purpose:** Isolated service for indexer configuration and search.

**Tech:** Go + Colly (scraping) + go-redis

### Network Access

```mermaid
graph TB
    subgraph "Search Gateway"
        Search[Search Gateway]
    end
    
    subgraph "Allowed Connections"
        A1[Redis: Cache results]
        A2[API Gateway: Receive queries]
        A3[Indexers: Outbound HTTP]
    end
    
    subgraph "Blocked"
        B1[Internet: Inbound]
        B2[PostgreSQL: No access]
        B3[MinIO: No access]
    end
    
    Search --> A1
    Search --> A2
    Search --> A3
    
    style B1 fill:#ff6666
    style B2 fill:#ff6666
    style B3 fill:#ff6666
```

### Indexer Types

```mermaid
graph TB
    subgraph "Supported Indexers"
        HTML[HTML Scrapers<br/>1337x, etc.]
        API[API-based<br/>Prowlarr, etc.]
        Custom[Custom<br/>User-defined]
    end
    
    subgraph "Gateway"
        Config[Config Manager]
        Search[Search Engine]
        Normalize[Result Normalizer]
    end
    
    Config --> HTML
    Config --> API
    Config --> Custom
    
    HTML --> Search
    API --> Search
    Custom --> Search
    
    Search --> Normalize
```

### Indexer Config Format

```json
{
  "indexers": [
    {
      "name": "1337x",
      "url": "https://1337x.to",
      "type": "html",
      "search_path": "/search/{query}/1/",
      "selectors": {
        "title": "td.name a:last-child",
        "url": "td.name a:last-child@href",
        "seeds": "td.seeds",
        "size": "td.size"
      }
    }
  ]
}
```

### Configuration

```yaml
CONFIG_PATH: /config/indexers.json
API_KEY: optional-api-key
REQUEST_TIMEOUT: 10
MAX_CONCURRENT: 5
```

### Go Project Structure

```
search/
├── cmd/
│   └── gateway/
│       └── main.go
├── internal/
│   ├── indexers/
│   │   ├── manager.go
│   │   ├── html.go
│   │   └── api.go
│   ├── search/
│   │   ├── engine.go
│   │   └── normalize.go
│   ├── cache/
│   │   └── redis.go
│   └── models/
│       ├── indexer.go
│       └── result.go
├── configs/
│   └── indexers.json
├── go.mod
└── go.sum
```

---

## Web Frontend

**Purpose:** User interface for the streaming service.

**Tech:** Vue + Vite + Tailwind + HLS.js

### Network Access

```mermaid
graph TB
    subgraph "Web App"
        Web[Web App]
    end
    
    subgraph "Allowed Connections"
        A1[Client: Accept HTTPS]
        A2[API Gateway: Forward requests]
    end
    
    subgraph "Blocked"
        B1[Direct service access]
    end
    
    Web --> A1
    Web --> A2
    
    style B1 fill:#ff6666
```

### Components

```mermaid
graph TB
    subgraph "Pages"
        Home[Home<br/>Browse]
        Login[Login]
        Library[Library<br/>Saved content]
        Media[Media<br/>Details + Player]
        Settings[Settings]
    end
    
    subgraph "Components"
        Player[Video Player<br/>HLS.js]
        Grid[Media Grid]
        Progress[Progress Bar]
        Format[Format Selector]
    end
    
    Home --> Grid
    Media --> Player
    Media --> Progress
    Media --> Format
    Library --> Grid
```

### Configuration

```yaml
NEXT_PUBLIC_API_URL: http://localhost:8000
NEXT_PUBLIC_WS_URL: ws://localhost:8000/ws
```

---

## Inter-Service Communication

### Task Queue (Asynq)

```mermaid
graph LR
    subgraph "Asynq (Redis)"
        Tasks[Task Queue<br/>Pending jobs]
        Active[Active<br/>Processing]
        Scheduled[Scheduled<br/>Delayed jobs]
        Retry[Retry<br/>Failed jobs]
    end
    
    API -->|Enqueue| Tasks
    Tasks -->|Dequeue| Active
    Active -->|Complete| Done[Done]
    Active -->|Fail| Retry
    Retry -->|Requeue| Tasks
    Worker -->|Process| Active
    Worker -->|Report| Progress[Progress<br/>Redis Pub/Sub]
    Progress -->|WebSocket| API
```

### Internal HTTP (API → Search)

```mermaid
sequenceDiagram
    participant A as API Gateway
    participant S as Search Gateway
    
    Note over A,S: Internal network only
    
    A->>S: POST /internal/search
    S->>S: Query indexers
    S->>A: Return results
```

### Asynq Task Types

```go
// Task types
const (
    TaskStream      = "stream:process"
    TaskArchive     = "archive:process"
    TaskCleanup     = "cleanup:segments"
    TaskHealthCheck = "worker:heartbeat"
)

// Task payload
type StreamTask struct {
    MediaID string
    UserID  string
    Save    bool
    Format  string // 1080p, 720p, 480p
}

// Enqueue example
client := asynq.NewClient(asynq.RedisClientOpt{Addr: "redis:6379"})
info, _ := client.Enqueue(
    asynq.NewTask(TaskStream, payload),
    asynq.Queue("streaming"),
    asynq.MaxRetry(3),
    asynq.Timeout(24*time.Hour),
)
```

### WebSocket Events

| Event | Direction | Data |
|-------|-----------|------|
| stream:progress | Server → Client | progress, speed, eta |
| stream:complete | Server → Client | hls_url, format |
| stream:error | Server → Client | error, recoverable |
| stream:buffering | Server → Client | progress, estimated_start |

### Asynq Monitor

Asynq provides a web UI for monitoring tasks:

```go
// In API gateway
mux.Handle("/asynq", asynqmonitor.NewHandler(monitorOpts))
```

Accessible at: `http://localhost:8000/asynq`

---

## Scaling Considerations

### Current Target (Self-Hosted)

- All services on one machine
- Docker Compose orchestration
- Single worker instance

### Future Growth

- Multiple worker instances
- Managed database
- Distributed MinIO
- Load balancer (if needed)
