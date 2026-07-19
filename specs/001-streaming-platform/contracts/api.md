# API Contracts: dotslashstream

**Base URL**: `http://localhost:8000`  
**Version**: v1  
**Auth**: JWT Bearer token

## Authentication

### Register

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}

Response: 201 Created
{
  "id": "uuid",
  "email": "user@example.com",
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Login

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "securepassword"
}

Response: 200 OK
{
  "access_token": "jwt-token",
  "token_type": "bearer",
  "expires_in": 3600
}
```

### Refresh

```http
POST /auth/refresh
Authorization: Bearer {refresh_token}

Response: 200 OK
{
  "access_token": "new-jwt-token",
  "token_type": "bearer",
  "expires_in": 3600
}
```

## Users

### Get Profile

```http
GET /users/me
Authorization: Bearer {token}

Response: 200 OK
{
  "id": "uuid",
  "email": "user@example.com",
  "preferences": {
    "default_format": "1080p",
    "auto_save": false
  },
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Update Profile

```http
PUT /users/me
Authorization: Bearer {token}
Content-Type: application/json

{
  "preferences": {
    "default_format": "720p",
    "auto_save": true
  }
}

Response: 200 OK
{
  "id": "uuid",
  "preferences": { ... },
  "updated_at": "2024-01-15T11:00:00Z"
}
```

## Media

### Get Media

```http
GET /media/{id}
Authorization: Bearer {token}

Response: 200 OK
{
  "id": "uuid",
  "type": "movie",
  "title": "Movie Title",
  "year": 2024,
  "poster_url": "https://...",
  "plot": "Description...",
  "formats": ["1080p", "720p", "480p"],
  "archived": true
}
```

### Search Media

```http
POST /search
Authorization: Bearer {token}
Content-Type: application/json

{
  "query": "movie title"
}

Response: 200 OK
{
  "results": [
    {
      "title": "Movie Title 2024",
      "torrent_url": "magnet:?xt=...",
      "indexer": "1337x",
      "size": 2147483648,
      "seeds": 150,
      "quality": "1080p"
    }
  ],
  "total_results": 25
}
```

### Stream Media

```http
POST /media/{id}/stream
Authorization: Bearer {token}
Content-Type: application/json

{
  "format": "1080p",
  "save": false
}

Response: 202 Accepted
{
  "stream_id": "uuid",
  "status": "buffering",
  "hls_url": null,
  "progress": 0
}
```

### Stream via Magnet

```http
POST /stream/magnet
Authorization: Bearer {token}
Content-Type: application/json

{
  "magnet_url": "magnet:?xt=urn:btih:...",
  "save": false
}

Response: 202 Accepted
{
  "stream_id": "uuid",
  "status": "buffering",
  "hls_url": null,
  "progress": 0
}
```

### Get Stream Progress

```http
GET /streams/{id}/progress
Authorization: Bearer {token}

Response: 200 OK
{
  "stream_id": "uuid",
  "status": "streaming",
  "progress": 45.2,
  "speed": "2.1 MB/s",
  "hls_url": "/hls/{stream_id}/master.m3u8"
}
```

### Stop Stream

```http
DELETE /streams/{id}
Authorization: Bearer {token}

Response: 204 No Content
```

## Library

### List Library

```http
GET /library
Authorization: Bearer {token}

Response: 200 OK
{
  "items": [
    {
      "id": "uuid",
      "media_id": "uuid",
      "title": "Movie Title",
      "type": "movie",
      "formats": ["1080p", "720p"],
      "total_size": 3221225472
    }
  ],
  "total": 50
}
```

### Save to Library

```http
POST /library
Authorization: Bearer {token}
Content-Type: application/json

{
  "media_id": "uuid",
  "format": "1080p"
}

Response: 202 Accepted
{
  "archive_id": "uuid",
  "status": "processing"
}
```

### Delete from Library

```http
DELETE /library/{id}
Authorization: Bearer {token}

Response: 204 No Content
```

## Playlists

### List Playlists

```http
GET /playlists
Authorization: Bearer {token}

Response: 200 OK
{
  "playlists": [
    {
      "id": "uuid",
      "name": "Weekend Movies",
      "item_count": 5,
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

### Create Playlist

```http
POST /playlists
Authorization: Bearer {token}
Content-Type: application/json

{
  "name": "My Watchlist"
}

Response: 201 Created
{
  "id": "uuid",
  "name": "My Watchlist",
  "item_count": 0
}
```

### Get Playlist

```http
GET /playlists/{id}
Authorization: Bearer {token}

Response: 200 OK
{
  "id": "uuid",
  "name": "My Watchlist",
  "items": [
    {
      "id": "uuid",
      "media_id": "uuid",
      "title": "Movie Title",
      "position": 0
    }
  ]
}
```

### Add to Playlist

```http
POST /playlists/{id}/items
Authorization: Bearer {token}
Content-Type: application/json

{
  "media_id": "uuid"
}

Response: 201 Created
{
  "id": "uuid",
  "position": 1
}
```

### Remove from Playlist

```http
DELETE /playlists/{id}/items/{item_id}
Authorization: Bearer {token}

Response: 204 No Content
```

## WebSocket

### Connect

```javascript
ws://localhost:8000/ws?token={jwt}
```

### Subscribe

```json
{
  "type": "subscribe",
  "channel": "stream:progress",
  "stream_id": "uuid"
}
```

### Events

```json
// Progress update
{
  "type": "stream:progress",
  "stream_id": "uuid",
  "progress": 45.2,
  "speed": "2.1 MB/s",
  "eta": 1800
}

// Stream ready
{
  "type": "stream:complete",
  "stream_id": "uuid",
  "hls_url": "/hls/{stream_id}/master.m3u8"
}

// Error
{
  "type": "stream:error",
  "stream_id": "uuid",
  "error": "No seeders available"
}
```

## Error Format

```json
{
  "error": {
    "code": "RESOURCE_NOT_FOUND",
    "message": "Media not found"
  }
}
```

### Error Codes

| Code | Status | Description |
|------|--------|-------------|
| AUTH_INVALID_CREDENTIALS | 401 | Invalid email/password |
| AUTH_TOKEN_EXPIRED | 401 | JWT expired |
| RESOURCE_NOT_FOUND | 404 | Resource doesn't exist |
| VALIDATION_ERROR | 422 | Invalid request body |
| RATE_LIMIT_EXCEEDED | 429 | Too many requests |
| STORAGE_QUOTA_EXCEEDED | 507 | Storage full |
| INDEXER_UNAVAILABLE | 502 | Search service down |
