# Feature Specification: dotslashstream Core Platform

**Feature Branch**: `001-streaming-platform`  
**Created**: 2024-01-18  
**Status**: Draft  
**Input**: User description: "create the spec based on the docs"

## Clarifications

### Session 2024-01-18

- Q: How do users add content to stream? → A: Both search and manual magnet/torrent link input
- Q: What content types should the system support? → A: Movies and TV shows only (no music)
- Q: What's the maximum concurrent downloads per user? → A: 3 concurrent downloads per user
- Q: How should TV shows be structured? → A: Season/Episode hierarchy (standard)
- Q: What level of logging/observability is required? → A: Structured logs with basic metrics

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Stream Media While Downloading (Priority: P1)

As a user, I want to start watching a video immediately after requesting it, without waiting for the full download to complete. The system should begin playing content as soon as enough data has been buffered.

**Why this priority**: This is the core value proposition of the platform. Without immediate streaming capability, the system is just another torrent client with extra steps.

**Independent Test**: Can be fully tested by requesting any media file and verifying playback starts within 10 seconds of request. Delivers immediate value as a streaming service.

**Acceptance Scenarios**:

1. **Given** a user is logged in and requests to stream media, **When** the torrent download starts, **Then** playback begins within 10 seconds once 5% of the file is buffered
2. **Given** a user is streaming media, **When** they pause and resume, **Then** playback continues from where they left off without re-downloading
3. **Given** a user requests media that is already partially downloaded, **When** they start streaming, **Then** the system uses cached data and starts playback faster

---

### User Story 2 - Save Media to Library (Priority: P2)

As a user, I want to save media I've watched to my permanent library in a specific quality format (1080p, 720p, or 480p) so I can access it later without re-downloading.

**Why this priority**: Library management is essential for a personal streaming service. Users need to build their collection over time.

**Independent Test**: Can be tested by requesting to save any media in a specific format and verifying it appears in the library with correct quality. Delivers value as a personal media server.

**Acceptance Scenarios**:

1. **Given** a user is streaming media, **When** they choose to save it in 1080p, **Then** the system streams immediately while archiving in the background
2. **Given** a user has saved media, **When** they access their library, **Then** they see all saved content with format options
3. **Given** a user wants to save media, **When** they select a format (1080p/720p/480p), **Then** the system transcodes to that quality and stores it permanently
4. **Given** a user has saved media, **When** they delete it, **Then** the system removes all associated files and frees storage space

---

### User Story 3 - Search for Media (Priority: P3)

As a user, I want to search for media across configured indexers so I can find content to stream or save.

**Why this priority**: Search is necessary to discover content, but the system can function with manually provided torrent links as a fallback.

**Independent Test**: Can be tested by performing a search query and verifying results are returned from configured indexers. Delivers value by enabling content discovery.

**Acceptance Scenarios**:

1. **Given** a user is logged in, **When** they enter a search query, **Then** results are returned from all configured indexers within 3 seconds
2. **Given** a user views search results, **When** they select a result, **Then** they can see details (quality, size, seeds) and choose to stream or save
3. **Given** indexers are configured, **When** a user searches, **Then** results are normalized across different indexer formats

---

### User Story 4 - User Authentication and Profiles (Priority: P4)

As a user, I want to create an account, log in, and manage my profile so my watch history, playlists, and preferences are preserved.

**Why this priority**: Authentication enables multi-user support and data persistence, but the core streaming works without it for single-user setups.

**Independent Test**: Can be tested by registering a new user, logging in, and verifying profile data persists across sessions.

**Acceptance Scenarios**:

1. **Given** a new user, **When** they register with email and password, **Then** an account is created and they can log in
2. **Given** a logged-in user, **When** they update preferences (default format, auto-save), **Then** settings persist across sessions
3. **Given** a user, **When** they log in on a new device, **Then** their library and watch history are accessible

---

### User Story 5 - Create and Manage Playlists (Priority: P5)

As a user, I want to create playlists to organize media I want to watch or have saved.

**Why this priority**: Playlists enhance organization but are not essential for core streaming functionality.

**Independent Test**: Can be tested by creating a playlist, adding media, and verifying the playlist persists and plays in order.

**Acceptance Scenarios**:

1. **Given** a logged-in user, **When** they create a playlist, **Then** it appears in their playlist list
2. **Given** a user has a playlist, **When** they add media items, **Then** items are added in the specified order
3. **Given** a user has a playlist, **When** they play it, **Then** media plays sequentially without manual intervention

---

### Edge Cases

- What happens when the torrent has no seeders? System should display an error and suggest trying again later
- What happens when storage is full during archiving? System should stop archiving and notify the user
- What happens when the user's session expires during streaming? System should maintain the stream and prompt re-authentication
- What happens when an indexer is unreachable? System should skip that indexer and return results from others
- What happens when transcoding fails? System should retry once, then mark the job as failed and notify the user
- What happens when the user tries to stream content that's still being archived? System should serve the temp stream until archive completes

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: System MUST allow users to register and authenticate with email/password
- **FR-002**: System MUST stream media content while it is being downloaded via torrent
- **FR-003**: System MUST transcode content to HLS format for adaptive bitrate streaming
- **FR-004**: System MUST support three archive quality profiles: 1080p, 720p, and 480p
- **FR-005**: System MUST provide a user library showing all saved media with format options
- **FR-006**: System MUST search across configured indexers and return normalized results
- **FR-021**: System MUST allow users to stream media by providing magnet links or torrent URLs directly
- **FR-022**: System MUST emit structured logs for all service operations
- **FR-023**: System MUST expose basic metrics (request count, active streams, storage usage)
- **FR-007**: System MUST allow users to create, edit, and delete playlists
- **FR-008**: System MUST track watch history and playback position per user
- **FR-009**: System MUST provide real-time download/transcoding progress via WebSocket
- **FR-010**: System MUST clean up temporary streaming segments after completion
- **FR-011**: System MUST isolate torrent worker from direct internet access
- **FR-012**: System MUST isolate search gateway from database and storage access
- **FR-013**: System MUST allow only the API Gateway to accept external connections
- **FR-014**: System MUST store all media in S3-compatible object storage
- **FR-015**: System MUST support optional original file retention after archiving
- **FR-016**: System MUST limit concurrent downloads to 3 per user
- **FR-017**: System MUST support configurable indexer definitions via JSON configuration
- **FR-018**: System MUST provide health check endpoints for all services
- **FR-019**: System MUST support rate limiting on API endpoints
- **FR-020**: System MUST handle torrent download failures with automatic retry

### Key Entities

- **User**: Person with account, preferences, and authentication credentials
- **Media**: Content item (movie or TV show) with metadata, torrent link, and available formats
- **Season**: Group of episodes within a TV show
- **Episode**: Individual TV show installment within a season
- **Playlist**: Named collection of media items in specific order
- **Stream**: Active streaming session with progress and status
- **Archive**: Permanently saved media in a specific format
- **Indexer**: External search source with configuration and normalization rules
- **Job**: Background task for downloading, transcoding, or cleanup operations
- **Segment**: Individual HLS chunk stored in object storage

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can start watching content within 10 seconds of requesting a stream
- **SC-002**: System supports at least 5 concurrent streaming sessions without performance degradation
- **SC-003**: 95% of search queries return results within 3 seconds
- **SC-004**: Archived content maintains quality indistinguishable from source at the selected profile
- **SC-005**: System recovers automatically from torrent download failures 90% of the time
- **SC-006**: Temporary streaming segments are cleaned up within 30 minutes of stream completion
- **SC-007**: Users can complete the save-to-library workflow in under 30 seconds of interaction time
- **SC-008**: System maintains 99.9% uptime for the API Gateway during normal operation
- **SC-009**: Network isolation prevents worker/search services from accepting external connections
- **SC-010**: All user data (preferences, playlists, watch history) persists across system restarts

## Assumptions

- Users have sufficient bandwidth for real-time streaming (minimum 5 Mbps recommended)
- The host system has sufficient CPU for at least 2 concurrent transcoding operations
- Torrent content is legally obtained or user has rights to stream/save
- Network infrastructure supports Docker container networking
- Storage capacity is sufficient for target library size (100GB+ recommended)
- Users will configure their own indexer sources
- Single-server deployment is the primary target (multi-server is future enhancement)
