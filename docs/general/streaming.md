# Streaming Pipeline

## Overview

HLS-based streaming enables playback during download. Content is transcoded on-the-fly and served as adaptive bitrate streams.

## Pipeline Architecture

```mermaid
graph TB
    subgraph "Input"
        Torrent[Torrent Client<br/>Sequential Mode]
    end
    
    subgraph "Processing"
        FFmpeg[FFmpeg<br/>Transcoder]
    end
    
    subgraph "Output"
        Temp[(temp/<br/>Transient HLS)]
        Archive[(archive/<br/>Permanent HLS)]
    end
    
    subgraph "Delivery"
        CDN[HLS Server]
        Player[Video Player]
    end
    
    Torrent -->|Sequential pieces| FFmpeg
    FFmpeg -->|Stream profile| Temp
    FFmpeg -->|Archive profile| Archive
    
    Temp -->|Stream only| CDN
    Archive -->|Saved content| CDN
    
    CDN -->|HLS segments| Player
```

## Streaming Modes

### Stream-Only

Fast transcoding for immediate playback. Segments are temporary.

```mermaid
sequenceDiagram
    participant T as Torrent
    participant F as FFmpeg
    participant S as Storage
    participant U as User
    
    T->>T: Download starts (sequential)
    Note over T: Wait for 5% buffer
    
    T->>F: Pipe data
    F->>F: Ultrafast transcode
    F->>S: Upload segments to temp/
    S->>U: Serve HLS stream
    
    Note over U: Watching...
    
    U->>U: Stream ends
    U->>S: Cleanup temp segments
```

**Profile:**
```yaml
video: h264 -crf 28 -preset ultrafast
audio: aac -b:a 128k
hls_time: 4
storage: temp/
```

### Archive Mode (Two-Phase)

Stream first, archive after download completes.

```mermaid
sequenceDiagram
    participant T as Torrent
    participant F as FFmpeg
    participant S as Storage
    participant U as User
    
    Note over T: Phase 1: Stream
    T->>F: Sequential data
    F->>S: Upload to temp/
    S->>U: HLS stream
    
    Note over T: Download complete
    
    Note over T: Phase 2: Archive
    T->>F: Complete file
    F->>F: Slow transcode (high quality)
    F->>S: Upload to archive/{id}/{format}/
    S->>S: Cleanup temp/
```

**Archive Profile (1080p):**
```yaml
video: h264 -crf 23 -preset slow -vf "scale=-2:1080"
audio: aac -b:a 192k
hls_time: 6
storage: archive/{media_id}/1080p/
```

## Transcoding Profiles

### Stream Profiles

```mermaid
graph LR
    subgraph "Stream-Only"
        S_Fast[ultrafast<br/>CRF 28<br/>128k audio]
    end
    
    subgraph "Use Case"
        S_Desc[Immediate playback<br/>Temporary storage<br/>Auto-cleanup]
    end
    
    S_Fast --> S_Desc
```

### Archive Profiles

```mermaid
graph TB
    subgraph "Archive Profiles"
        A_1080[1080p<br/>CRF 23<br/>192k audio]
        A_720[720p<br/>CRF 26<br/>128k audio]
        A_480[480p<br/>CRF 30<br/>96k audio]
    end
    
    subgraph "Characteristics"
        C_High[High quality<br/>Larger files<br/>Slow encode]
        C_Mid[Balanced<br/>Medium files<br/>Medium encode]
        C_Low[Space efficient<br/>Smaller files<br/>Slow encode]
    end
    
    A_1080 --> C_High
    A_720 --> C_Mid
    A_480 --> C_Low
```

| Profile | Resolution | CRF | Preset | Audio | HLS Time |
|---------|------------|-----|--------|-------|----------|
| stream-only | Original | 28 | ultrafast | 128k | 4s |
| archive-1080p | 1920x1080 | 23 | slow | 192k | 6s |
| archive-720p | 1280x720 | 26 | slow | 128k | 6s |
| archive-480p | 854x480 | 30 | slow | 96k | 6s |

## Torrent Configuration

### Sequential Download

For streaming, pieces must be downloaded in order.

```mermaid
graph TB
    subgraph "Normal Mode"
        N[Piece 5<br/>Piece 12<br/>Piece 3<br/>Piece 8]
    end
    
    subgraph "Sequential Mode"
        S[Piece 1<br/>Piece 2<br/>Piece 3<br/>Piece 4]
    end
    
    subgraph "FFmpeg Requirement"
        R[Needs sequential data<br/>for real-time decode]
    end
    
    N -->|Not suitable| R
    S -->|Perfect for streaming| R
```

### Priority Management

```mermaid
graph LR
    subgraph "Piece Priority"
        P1[First 5%<br/>Priority: Highest]
        P2[Rest<br/>Priority: Normal]
    end
    
    subgraph "Goal"
        G[Start streaming<br/>as fast as possible]
    end
    
    P1 --> G
    P2 --> G
```

## HLS Structure

### Master Playlist

```mermaid
graph TB
    Master[master.m3u8<br/>Master Playlist]
    
    Master --> V1080[1080p/master.m3u8<br/>2 Mbps]
    Master --> V720[720p/master.m3u8<br/>1 Mbps]
    Master --> V480[480p/master.m3u8<br/>500 kbps]
    
    V1080 --> Seg1080[segment_000.ts<br/>segment_001.ts<br/>...]
    V720 --> Seg720[segment_000.ts<br/>segment_001.ts<br/>...]
    V480 --> Seg480[segment_000.ts<br/>segment_001.ts<br/>...]
```

### Segment Lifecycle

```mermaid
stateDiagram-v2
    [*] --> Created: FFmpeg output
    
    Created --> Uploaded: Upload to S3
    
    Uploaded --> Serving: Player requests
    
    Serving --> Active: Ongoing stream
    
    Active --> Expired: Stream ends
    
    Expired --> Deleted: Cleanup job
    
    Deleted --> [*]
    
    note right of Serving
        Stream-only: 30min TTL
        Archive: Permanent
    end note
```

## Buffer Management

### Initial Buffer Strategy

```mermaid
graph TB
    subgraph "Buffer Requirements"
        B1[Minimum 5% downloaded]
        B2[Minimum 10 seconds]
    end
    
    subgraph "Decision"
        D{Buffer ready?}
    end
    
    subgraph "Actions"
        A1[Yes: Start FFmpeg]
        A2[No: Continue downloading]
    end
    
    B1 --> D
    B2 --> D
    
    D -->|Yes| A1
    D -->|No| A2
    
    A2 --> D
```

### Adaptive Buffer

```go
const (
    MinBufferPercent = 0.05  // 5% of total size
    MinBufferSeconds = 10    // minimum 10 seconds of content
)

func checkBufferReady(torrent *Torrent, mediaDuration float64) bool {
    downloadedPercent := torrent.Progress()
    downloadedSeconds := mediaDuration * downloadedPercent
    
    return downloadedPercent >= MinBufferPercent &&
           downloadedSeconds >= MinBufferSeconds
}
```

## Error Handling

### Transcoding Failures

```mermaid
graph TB
    subgraph "Error Types"
        E1[Invalid data<br/>Corrupted input]
        E2[No space left<br/>Storage full]
        E3[Unknown error<br/>Generic failure]
    end
    
    subgraph "Recovery"
        R1[Retry download]
        R2[Cleanup old streams<br/>Retry transcode]
        R3[Notify user<br/>Mark failed]
    end
    
    E1 --> R1
    E2 --> R2
    E3 --> R3
```

### Network Interruptions

```mermaid
graph LR
    subgraph "Disconnect Detected"
        D[Stream interrupted]
    end
    
    subgraph "Recovery"
        K[Keep segments for 5min]
        R[Allow reconnection]
        C[Cleanup after timeout]
    end
    
    D --> K
    K --> R
    K --> C
```

## Performance Tuning

### CPU Optimization

```yaml
# Limit FFmpeg threads per job
threads: 2

# Max concurrent transcodes
max_concurrent: 2

# GPU acceleration (if available)
hwaccel: vaapi  # or cuda, videotoolbox
```

### Memory Management

```yaml
# FFmpeg buffer size
buffer_size: 10M

# Segment memory limit
segment_buffer: 50M
```

## Monitoring Metrics

| Metric | Description | Target |
|--------|-------------|--------|
| start_to_first_segment | Time to first playable segment | < 10s |
| transcode_speed | Ratio of realtime | > 1.5x |
| segment_generation_rate | Segments per second | > 10 |
| error_rate | Failed transcodes per hour | < 1 |
