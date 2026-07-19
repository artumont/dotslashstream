# Storage Architecture

## Overview

All media storage uses S3-compatible object storage (MinIO). Only API Gateway and Torrent Worker have access.

## Bucket Structure

```mermaid
graph TB
    subgraph "media-bucket"
        subgraph "temp/"
            T1[session_abc/<br/>master.m3u8<br/>segment_000.ts<br/>segment_001.ts]
        end
        
        subgraph "archive/"
            subgraph "{media_id}/"
                A1[1080p/<br/>master.m3u8<br/>segments...]
                A2[720p/<br/>master.m3u8<br/>segments...]
                A3[metadata.json]
            end
        end
        
        subgraph "originals/"
            O1["{media_id}.mp4"]
        end
    end
    
    style temp fill:#ff9999
    style archive fill:#99ff99
    style originals fill:#9999ff
```

## Storage Access

### Who Can Access

```mermaid
graph TB
    subgraph "MinIO Access"
        API[API Gateway]
        Worker[Torrent Worker]
        MinIO[MinIO]
    end
    
    subgraph "Permissions"
        P1[API: Read metadata, generate URLs]
        P2[Worker: Upload/download segments]
    end
    
    API -->|Read/Write| MinIO
    Worker -->|Read/Write| MinIO
    
    API --- P1
    Worker --- P2
    
    style Worker fill:#ff9999
```

### Isolation Rules

| Service | Access | Operations |
|---------|--------|------------|
| API Gateway | ✅ | Read metadata, generate presigned URLs |
| Torrent Worker | ✅ | Upload segments, download for re-transcode |
| Search Gateway | ❌ | No access |
| Web App | ❌ | No direct access (via API only) |

## Storage Classes

### Temporary (`temp/`)

**Purpose:** HLS segments for active streams

```mermaid
stateDiagram-v2
    [*] --> Created: Stream starts
    
    Created --> Populating: Segments uploaded
    
    Populating --> Active: Serving player
    
    Active --> Cleaning: Stream ends
    
    Cleaning --> Deleted: Cleanup job
    
    Deleted --> [*]
    
    note right of Active
        Max age: 24h
        Idle timeout: 30m
        Max size: 10GB
    end note
```

### Archive (`archive/`)

**Purpose:** Permanent media library

**Movie Structure:**
```mermaid
graph TB
    subgraph "Movie: {media_id}"
        M[metadata.json]
        
        subgraph "Formats"
            F1080[1080p/]
            F720[720p/]
            F480[480p/]
        end
    end
    
    M --> F1080
    M --> F720
    M --> F480
```

**TV Show Structure:**
```mermaid
graph TB
    subgraph "TV Show: {media_id}"
        M[metadata.json]
        
        subgraph "Season 1"
            S1M[season_metadata.json]
            E1[episode_01/]
            E2[episode_02/]
        end
        
        subgraph "Season 2"
            S2M[season_metadata.json]
            E3[episode_01/]
        end
    end
    
    M --> S1M
    M --> S2M
    S1M --> E1
    S1M --> E2
    S2M --> E3
```

**Metadata Example (Movie):**
```json
{
  "media_id": "abc123",
  "type": "movie",
  "title": "Movie Title",
  "year": 2024,
  "formats": {
    "1080p": { "path": "1080p/", "size": 2147483648, "duration": 7200 },
    "720p": { "path": "720p/", "size": 1073741824, "duration": 7200 }
  }
}
```

**Metadata Example (TV Show):**
```json
{
  "media_id": "xyz789",
  "type": "tv",
  "title": "Show Title",
  "seasons": [
    {
      "number": 1,
      "episodes": [
        { "number": 1, "title": "Pilot", "formats": { "1080p": {}, "720p": {} } },
        { "number": 2, "title": "Episode 2", "formats": { "1080p": {} } }
      ]
    }
  ]
}
```

### Originals (`originals/`)

**Purpose:** Keep source files (optional)

```mermaid
graph LR
    subgraph "Config"
        C{Keep originals?}
    end
    
    subgraph "Actions"
        A1[Yes: Store in originals/]
        A2[No: Delete after archive]
    end
    
    C -->|true| A1
    C -->|false| A2
```

## MinIO Configuration

### Docker Setup

```mermaid
graph TB
    subgraph "MinIO Container"
        API["S3 API<br/>Port 9000"]
        Console["Web Console<br/>Port 9001"]
        Data["/data<br/>Storage"]
    end
    
    subgraph "Host"
        Mount["/mnt/media<br/>Large storage"]
    end
    
    Mount --> Data
```

### Network Access

```yaml
networks:
  storage:
    driver: bridge
    internal: true  # No external access

services:
  minio:
    networks:
      - storage
    # Only API and Worker can reach this
```

### Environment

```yaml
MINIO_ROOT_USER: admin
MINIO_ROOT_PASSWORD: secure-password
MINIO_BUCKET: media
MINIO_ENDPOINT: minio:9000
MINIO_ACCESS_KEY: admin
MINIO_SECRET_KEY: secure-password
```

## S3 Client Operations

### Core Operations

```mermaid
graph TB
    subgraph "Client Operations"
        Upload[Upload File]
        Download[Download File]
        List[List Objects]
        Delete[Delete Object]
        URL[Presigned URL]
    end
    
    subgraph "Use Cases"
        U1[Worker: Upload segments]
        U2[API: Generate stream URL]
        U3[Cleanup: Delete old segments]
        U4[Backup: List all objects]
    end
    
    Upload --> U1
    URL --> U2
    Delete --> U3
    List --> U4
```

### Presigned URLs

```mermaid
sequenceDiagram
    participant C as Client
    participant A as API
    participant S as MinIO
    
    C->>A: Request stream URL
    A->>A: Generate presigned URL
    A->>C: Return URL (expires in 1h)
    
    C->>S: Request HLS segment
    S->>C: Serve segment
```

### Go Client

```go
package storage

import (
    "context"
    "github.com/minio/minio-go/v7"
)

type MinIOClient struct {
    client *minio.Client
    bucket string
}

func NewMinIOClient(endpoint, accessKey, secretKey, bucket string) (*MinIOClient, error) {
    client, err := minio.New(endpoint, &minio.Options{
        Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
        Secure: false,
    })
    if err != nil {
        return nil, err
    }
    
    return &MinIOClient{
        client: client,
        bucket: bucket,
    }, nil
}

func (m *MinIOClient) UploadSegment(ctx context.Context, key string, reader io.Reader, size int64) error {
    _, err := m.client.PutObject(ctx, m.bucket, key, reader, size, minio.PutObjectOptions{
        ContentType: "video/mp2t",
    })
    return err
}

func (m *MinIOClient) GetPresignedURL(ctx context.Context, key string, expiry time.Duration) (string, error) {
    reqParams := url.Values{}
    presignedURL, err := m.client.PresignedGetObject(ctx, m.bucket, key, expiry, reqParams)
    if err != nil {
        return "", err
    }
    return presignedURL.String(), nil
}
```

## Storage Lifecycle

### Automatic Cleanup

```mermaid
graph TB
    subgraph "Cleanup Jobs"
        J1[Hourly: Remove old temp segments]
        J2[Daily: Check storage usage]
        J3[Weekly: Verify archive integrity]
    end
    
    subgraph "Triggers"
        T1[Age > 24h]
        T2[Idle > 30m]
        T3[Size > 10GB]
    end
    
    subgraph "Actions"
        A1[Delete segments]
        A2[Log warning]
        A3[Send alert]
    end
    
    J1 --> T1
    J1 --> T2
    J2 --> T3
    
    T1 --> A1
    T2 --> A1
    T3 --> A2
    T3 --> A3
```

### Manual Cleanup

```mermaid
graph LR
    subgraph "User Action"
        U[Delete media]
    end
    
    subgraph "System"
        S[Check formats]
        D[Delete specified]
        O[Delete originals if enabled]
        F[Log freed space]
    end
    
    U --> S
    S --> D
    D --> O
    O --> F
```

## Backup Strategy

### Backup Flow

```mermaid
graph TB
    subgraph "Backup Process"
        Stop[Stop services]
        DB[Backup PostgreSQL]
        S3[Backup MinIO]
        Config[Backup configs]
        Start[Start services]
    end
    
    Stop --> DB
    DB --> S3
    S3 --> Config
    Config --> Start
```

### Restore Flow

```mermaid
graph TB
    subgraph "Restore Process"
        Stop[Stop services]
        DB[Restore PostgreSQL]
        S3[Restore MinIO]
        Config[Restore configs]
        Start[Start services]
        Verify[Verify integrity]
    end
    
    Stop --> DB
    DB --> S3
    S3 --> Config
    Config --> Start
    Start --> Verify
```

## Migration to Cloud S3

```mermaid
graph TB
    subgraph "Current"
        MinIO[MinIO<br/>Self-hosted]
    end
    
    subgraph "Migration"
        Sync[mc mirror<br/>Sync data]
        Update[Update config]
        Test[Test operations]
    end
    
    subgraph "Target"
        S3[AWS S3<br/>Cloud]
    end
    
    MinIO --> Sync
    Sync --> Update
    Update --> Test
    Test --> S3
```

**Steps:**
1. Update environment variables
2. Sync data: `mc mirror media-bucket/ s3/media-bucket/`
3. Update application configuration
4. Test all operations
5. Decommission MinIO

## Storage Monitoring

### Key Metrics

| Metric | Description | Alert Threshold |
|--------|-------------|-----------------|
| bucket_size_total | Total bytes in bucket | > 500GB |
| bucket_size_temp | Temp storage usage | > 10GB |
| object_count | Number of objects | > 10000 |
| upload_bandwidth | Upload speed | < 10MB/s |
| download_bandwidth | Download speed | < 50MB/s |
| error_rate | Failed requests | > 1% |

### Health Check

```mermaid
graph TB
    subgraph "Health Check"
        H[Check connection]
        S[Get bucket stats]
        V[Verify access]
    end
    
    subgraph "Output"
        O["status: healthy, bucket: media,
object_count: 150, total_size: 2.3GB"]
    end
    
    H --> S
    S --> V
    V --> O
```
