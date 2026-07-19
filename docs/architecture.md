# Architecture

## Overview

dotslashstream is a self-hosted media streaming service with four core services. API Gateway is the single entry point — all other services are network isolated.

## Design Principles

1. **Single entry point** — Only API Gateway exposes ports to the network
2. **Network isolation** — Workers and search gateway have no internet access
3. **Service isolation** — Compromised service can't affect others
4. **Stream first** — Media plays as soon as initial buffer is ready
5. **Save optionally** — Users choose whether to archive content
6. **Object storage** — All media stored in S3-compatible buckets
7. **Liability isolation** — Indexer configs live on separate gateway
8. **Flexible input** — Content via search or manual magnet/torrent links

## System Diagram

```mermaid
graph TB
    subgraph "External Network"
        Client[Client<br/>Browser]
    end
    
    subgraph "DMZ - Exposed"
        Web[Web App]
        API[API Gateway<br/>Go]
    end
    
    subgraph "Internal Network - Isolated"
        Worker[Torrent Worker<br/>Go]
        Search[Search Gateway<br/>Go]
    end
    
    subgraph "Storage Layer"
        DB[(PostgreSQL)]
        Queue[(Redis + Asynq)]
        S3[(MinIO)]
    end
    
    Client -->|HTTPS| Web
    Client -->|WSS| API
    Web -->|HTTP| API
    
    API -->|Enqueue jobs| Queue
    API -->|Internal only| DB
    API -->|Internal only| Search
    
    Queue -->|Process jobs| Worker
    Worker -->|Internal only| S3
    Worker -->|Report progress| Queue
    
    style Worker fill:#ff9999
    style Search fill:#ff9999
```

## Network Isolation

### Exposed Services (DMZ)

| Service | Ports | Protocol | Purpose |
|---------|-------|----------|---------|
| API Gateway | 8000, 8443 | HTTP/HTTPS | REST API, WebSocket |
| Web App | 3000, 443 | HTTPS | Frontend UI |

### Isolated Services (Internal)

| Service | Ports | Protocol | Purpose |
|---------|-------|----------|---------|
| Torrent Worker | None | Internal only | Download, transcode |
| Search Gateway | None | Internal only | Indexer queries |

### Isolation Rules

```mermaid
graph TB
    subgraph "Firewall Rules"
        R1[Allow: Client → Web:443]
        R2[Allow: Client → API:8443]
        R3[Allow: Web → API:8000]
        R4[Deny: All other inbound]
        R5[Deny: Worker → Internet]
        R6[Deny: Search → Internet]
    end
    
    subgraph "Internal Only"
        I1[API → Worker: via Redis]
        I2[API → Search: internal HTTP]
        I3[Worker → S3: internal HTTP]
        I4[Worker → Cache: internal Redis]
    end
    
    R1 --> R4
    R2 --> R4
    R3 --> R4
    R5 --> R4
    R6 --> R4
```

### Docker Network Configuration

```yaml
networks:
  dmz:
    driver: bridge
    ipam:
      config:
        - subnet: 172.20.0.0/24
  
  internal:
    driver: bridge
    internal: true  # No external access
    ipam:
      config:
        - subnet: 172.21.0.0/24
  
  storage:
    driver: bridge
    internal: true
    ipam:
      config:
        - subnet: 172.22.0.0/24

services:
  api:
    networks:
      - dmz
      - internal
      - storage
  
  web:
    networks:
      - dmz
  
  worker:
    networks:
      - internal
      - storage
    # No dmz network = no internet
  
  search:
    networks:
      - internal
    # No dmz network = no internet
  
  db:
    networks:
      - storage
  
  redis:
    networks:
      - internal
      - storage
  
  minio:
    networks:
      - storage
```

## Attack Surface Reduction

### Worker Isolation

```mermaid
graph TB
    subgraph "Worker - No External Access"
        W[Torrent Worker]
        
        subgraph "Allowed"
            A1[Redis: Read jobs, write progress]
            A2[MinIO: Upload/download segments]
        end
        
        subgraph "Blocked"
            B1[Internet]
            B2[Database]
            B3[User traffic]
        end
    end
    
    W --> A1
    W --> A2
    
    style B1 fill:#ff6666
    style B2 fill:#ff6666
    style B3 fill:#ff6666
```

**Worker can:**
- Consume jobs from Redis
- Download torrents (via internal proxy if needed)
- Upload segments to MinIO
- Publish progress to Redis

**Worker cannot:**
- Access internet directly
- Access database
- Accept inbound connections
- Execute arbitrary code from external sources

### Search Gateway Isolation

```mermaid
graph TB
    subgraph "Search Gateway - No External Access"
        S[Search Gateway]
        
        subgraph "Allowed"
            A1[Redis: Cache results]
            A2[Internal: Receive queries from API]
        end
        
        subgraph "Blocked"
            B1[Internet]
            B2[Database]
            B3[MinIO]
        end
    end
    
    S --> A1
    S --> A2
    
    style B1 fill:#ff6666
    style B2 fill:#ff6666
    style B3 fill:#ff6666
```

**Search Gateway can:**
- Receive queries from API (internal network)
- Query configured indexers (outbound HTTP)
- Cache results in Redis

**Search Gateway cannot:**
- Accept connections from outside
- Access database
- Access MinIO
- Modify system configuration

## Data Flow

### Stream Request

```mermaid
sequenceDiagram
    participant C as Client
    participant W as Web App
    participant A as API Gateway
    participant Q as Redis
    participant T as Torrent Worker
    participant S as MinIO
    
    C->>W: Request stream
    W->>A: POST /media/{id}/stream
    A->>Q: Publish job
    
    Note over Q,T: Internal network only
    
    Q->>T: Consume job
    T->>T: Start torrent (sequential)
    T->>T: Wait for 5% buffer
    
    loop Streaming
        T->>T: FFmpeg transcode to HLS
        T->>S: Upload segments
        T->>Q: Publish progress
        Q->>A: Forward progress
        A->>W: WebSocket update
        W->>C: Stream + progress
    end
    
    T->>A: Stream complete
    A->>W: Ready notification
```

### Search Request

```mermaid
sequenceDiagram
    participant C as Client
    participant W as Web App
    participant A as API Gateway
    participant SG as Search Gateway
    participant I as Indexers
    
    C->>W: Search query
    W->>A: POST /search
    A->>SG: Forward query (internal)
    
    Note over SG,I: Isolated network
    
    SG->>I: Query indexers
    I->>SG: Return results
    SG->>A: Normalized results
    A->>W: Response
    W->>C: Display results
```

## Service Responsibilities

### API Gateway (Go)

- JWT authentication
- User profiles and preferences
- Playlist management
- Job queue coordination
- WebSocket hub for real-time updates
- Rate limiting
- Request validation

### Torrent Worker (Go)

- Torrent lifecycle management
- Sequential piece delivery
- FFmpeg transcoding pipeline
- Object storage integration
- Progress reporting

### Search Gateway (Go)

- Indexer configuration management
- Parallel search across indexers
- Result normalization
- Liability isolation

## Security Layers

```mermaid
graph TB
    subgraph "Layer 1: Network"
        N1[Firewall rules]
        N2[Docker networks]
        N3[No exposed ports]
    end
    
    subgraph "Layer 2: Service"
        S1[Minimal permissions]
        S2[No shared users]
        S3[Read-only filesystems]
    end
    
    subgraph "Layer 3: Application"
        A1[Input validation]
        A2[Authentication]
        A3[Rate limiting]
    end
    
    subgraph "Layer 4: Data"
        D1[Encryption at rest]
        D2[Encryption in transit]
        D3[Secret management]
    end
    
    N1 --> S1
    S1 --> A1
    A1 --> D1
```

## Component Details

See [services.md](./services.md) for detailed service specifications.

## Storage Architecture

See [storage.md](./storage.md) for bucket structure and lifecycle.

## Streaming Pipeline

See [streaming.md](./streaming.md) for transcoding details.
