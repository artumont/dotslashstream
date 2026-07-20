# Tasks: dotslashstream Core Platform

**Input**: Design documents from `/specs/001-streaming-platform/`
**Prerequisites**: plan.md, spec.md, research.md, data-model.md, contracts/api.md

**Tests**: Not explicitly requested in spec. Skipping test tasks.

**Organization**: Tasks grouped by user story for independent implementation.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (US1-US5)
- Exact file paths included in descriptions

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization for 4 services + frontend

- [ ] T001 Create project structure per plan.md (api/, worker/, search/, web/, docker-compose.yml)
- [ ] T002 [P] Initialize Go module for api service with dependencies (net/http, asynq, sqlc, jwt-go)
- [ ] T003 [P] Initialize Go module for worker service with dependencies (anacrolix/torrent, minio-go, asynq)
- [ ] T004 [P] Initialize Go module for search service with dependencies (colly, redis)
- [ ] T005 [P] Initialize Vue.js project in web/ with Vite, Pinia, Vue Router
- [ ] T006 Create docker-compose.yml with services (api, worker, search, web, postgres, redis, minio)
- [ ] T007 Create .env.example with all required environment variables
- [ ] T008 [P] Create api/cmd/server/main.go entry point with basic HTTP server
- [ ] T009 [P] Create worker/cmd/worker/main.go entry point with asynq worker
- [ ] T010 [P] Create search/cmd/gateway/main.go entry point with HTTP server
- [ ] T011 [P] Create web/src/App.vue and web/src/main.js entry points

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that ALL user stories depend on

**⚠️ CRITICAL**: No user story work until this phase completes

- [ ] T012 Create PostgreSQL migrations in api/migrations/ (users, media, season, episode, stream, archive, playlist, playlist_item, watch_history, indexer, job, segment tables)
- [ ] T013 [P] Configure sqlc in api/ with queries.sql for all entities
- [ ] T014 [P] Create Redis connection helper in api/internal/cache/redis.go
- [ ] T015 [P] Create MinIO client setup in worker/internal/storage/minio.go
- [ ] T016 Implement JWT authentication in api/internal/auth/jwt.go (generate, validate, refresh tokens)
- [ ] T017 Implement auth middleware in api/internal/auth/middleware.go
- [ ] T018 Create API router with middleware chain in api/cmd/server/main.go
- [ ] T019 [P] Implement error handling and structured logging in api/internal/logger/
- [ ] T020 [P] Create WebSocket hub for real-time updates in api/internal/websocket/hub.go
- [ ] T021 [P] Create asynq client and queue setup in worker/internal/queue/asynq.go

**Checkpoint**: Foundation ready. All services can connect to dependencies, auth works, routing established.

---

## Phase 3: User Story 1 - Stream Media While Downloading (Priority: P1) 🎯 MVP

**Goal**: User can request media via magnet link and start watching within 10 seconds

**Independent Test**: Provide magnet link → torrent downloads sequentially → HLS segments generated → playback starts in browser

### Implementation for User Story 1

- [ ] T022 [P] [US1] Create Stream model and repository in api/internal/models/stream.go and api/internal/repository/stream.go
- [ ] T023 [P] [US1] Create Job model and repository in api/internal/models/job.go and api/internal/repository/job.go
- [ ] T024 [US1] Implement torrent client with sequential download in worker/internal/torrent/client.go
- [ ] T025 [US1] Implement torrent download handler in worker/internal/torrent/download.go (sequential piece priority, progress reporting)
- [ ] T026 [US1] Implement FFmpeg HLS transcoding pipeline in worker/internal/transcode/ffmpeg.go (pipe stdin, output segments)
- [ ] T027 [US1] Implement HLS segment generation in worker/internal/transcode/hls.go (master.m3u8, .ts segments)
- [ ] T028 [US1] Create stream processing job handler in worker/internal/queue/asynq.go (task type: stream:process)
- [ ] T029 [US1] Create StreamService in api/internal/service/stream.go (create stream, track progress, get HLS URL)
- [ ] T030 [US1] Implement POST /stream/magnet endpoint in api/internal/handlers/stream.go
- [ ] T031 [US1] Implement GET /streams/{id}/progress endpoint in api/internal/handlers/stream.go
- [ ] T032 [US1] Implement DELETE /streams/{id} endpoint in api/internal/handlers/stream.go
- [ ] T033 [US1] Implement WebSocket stream:progress events in api/internal/handlers/stream.go
- [ ] T034 [US1] Create MinIO presigned URL generation for HLS segments in worker/internal/storage/minio.go
- [ ] T035 [US1] Create VideoPlayer.vue component in web/src/components/player/VideoPlayer.vue (HLS.js integration)
- [ ] T036 [US1] Create stream store in web/src/stores/player.js (connect WebSocket, manage playback state)

**Checkpoint**: MVP complete. User can stream any torrent content immediately.

---

## Phase 4: User Story 2 - Save Media to Library (Priority: P2)

**Goal**: User can permanently save streamed media in specific quality (1080p/720p/480p)

**Independent Test**: Stream media → choose save format → archive completes → media appears in library with correct format

### Implementation for User Story 2

- [ ] T037 [P] [US2] Create Media model and repository in api/internal/models/media.go and api/internal/repository/media.go
- [ ] T038 [P] [US2] Create Archive model and repository in api/internal/models/archive.go
- [ ] T039 [US2] Create quality transcoding profiles in worker/internal/transcode/profiles.go (1080p, 720p, 480p presets)
- [ ] T040 [US2] Create archive processing job handler in worker/internal/queue/asynq.go (task type: archive:process)
- [ ] T041 [US2] Create MediaService in api/internal/service/media.go (get media, list library)
- [ ] T042 [US2] Implement POST /library endpoint in api/internal/handlers/media.go (save to library)
- [ ] T043 [US2] Implement GET /library endpoint in api/internal/handlers/media.go (list saved media)
- [ ] T044 [US2] Implement DELETE /library/{id} endpoint in api/internal/handlers/media.go (remove from library)
- [ ] T045 [US2] Implement GET /media/{id} endpoint in api/internal/handlers/media.go (get media details)
- [ ] T046 [US2] Create MediaCard.vue component in web/src/components/media/MediaCard.vue
- [ ] T047 [US2] Create MediaGrid.vue component in web/src/components/media/MediaGrid.vue
- [ ] T048 [US2] Create Library.vue view in web/src/views/Library.vue
- [ ] T049 [US2] Create media store in web/src/stores/media.js (library management)
- [ ] T050 [US2] Integrate save option into VideoPlayer.vue (show save button during streaming)

**Checkpoint**: Users can build permanent media library with quality selection.

---

## Phase 5: User Story 3 - Search for Media (Priority: P3)

**Goal**: User can search across configured indexers to find content

**Independent Test**: Configure indexer → search query → results returned within 3 seconds → can stream or save from results

### Implementation for User Story 3

- [ ] T051 [P] [US3] Create Indexer model and repository in api/internal/models/indexer.go
- [ ] T052 [US3] Create indexer manager in search/internal/indexers/manager.go (load config, manage indexers)
- [ ] T053 [US3] Implement HTML indexer scraper in search/internal/indexers/html.go (Colly-based, configurable selectors)
- [ ] T054 [US3] Implement API indexer client in search/internal/indexers/api.go (REST API integration)
- [ ] T055 [US3] Create search engine in search/internal/search/engine.go (fan-out to indexers, merge results)
- [ ] T056 [US3] Implement result normalization in search/internal/search/normalize.go (standardize across indexers)
- [ ] T057 [US3] Create Redis cache for search results in search/internal/cache/redis.go
- [ ] T058 [US3] Create search/indexers.json configuration file in search/configs/indexers.json
- [ ] T059 [US3] Implement POST /search endpoint in api/internal/handlers/search.go (proxy to search gateway)
- [ ] T060 [US3] Create Search.vue view in web/src/views/Search.vue (search input, results grid)
- [ ] T061 [US3] Add search results to media store in web/src/stores/media.js

**Checkpoint**: Content discovery functional across multiple indexers.

---

## Phase 6: User Story 4 - User Authentication and Profiles (Priority: P4)

**Goal**: Full user registration, login, profile management, and watch history

**Independent Test**: Register new user → login → update preferences → watch history persists → logout → login again preserves data

### Implementation for User Story 4

- [ ] T062 [P] [US4] Create UserRepository in api/internal/repository/user.go (CRUD operations)
- [ ] T063 [US4] Create AuthService in api/internal/service/auth.go (register, login, refresh)
- [ ] T064 [US4] Create UserService in api/internal/service/user.go (get/update profile)
- [ ] T065 [US4] Implement POST /auth/register endpoint in api/internal/handlers/auth.go
- [ ] T066 [US4] Implement POST /auth/login endpoint in api/internal/handlers/auth.go
- [ ] T067 [US4] Implement POST /auth/refresh endpoint in api/internal/handlers/auth.go
- [ ] T068 [US4] Implement GET /users/me endpoint in api/internal/handlers/users.go
- [ ] T069 [US4] Implement PUT /users/me endpoint in api/internal/handlers/users.go
- [ ] T070 [P] [US4] Create WatchHistory model and repository in api/internal/models/watch_history.go
- [ ] T071 [US4] Add watch history tracking to stream completion in api/internal/service/stream.go
- [ ] T072 [US4] Create Login.vue view in web/src/views/Login.vue (login/register forms)
- [ ] T073 [US4] Create Settings.vue view in web/src/views/Settings.vue (profile, preferences)
- [ ] T074 [US4] Create auth store in web/src/stores/auth.js (token management, user state)
- [ ] T075 [US4] Add route guards for authenticated routes in web/src/router/

**Checkpoint**: Multi-user support with persistent profiles and history.

---

## Phase 7: User Story 5 - Create and Manage Playlists (Priority: P5)

**Goal**: User can create playlists and organize media for sequential playback

**Independent Test**: Create playlist → add media → play playlist → media plays in order → remove item → delete playlist

### Implementation for User Story 5

- [ ] T076 [P] [US5] Create Playlist model and repository in api/internal/models/playlist.go
- [ ] T077 [P] [US5] Create PlaylistItem model and repository in api/internal/models/playlist_item.go
- [ ] T078 [US5] Create PlaylistService in api/internal/service/playlist.go (CRUD, add/remove items, reorder)
- [ ] T079 [US5] Implement GET /playlists endpoint in api/internal/handlers/playlists.go
- [ ] T080 [US5] Implement POST /playlists endpoint in api/internal/handlers/playlists.go
- [ ] T081 [US5] Implement GET /playlists/{id} endpoint in api/internal/handlers/playlists.go
- [ ] T082 [US5] Implement POST /playlists/{id}/items endpoint in api/internal/handlers/playlists.go
- [ ] T083 [US5] Implement DELETE /playlists/{id}/items/{item_id} endpoint in api/internal/handlers/playlists.go
- [ ] T084 [US5] Create PlaylistList.vue component in web/src/components/playlist/PlaylistList.vue
- [ ] T085 [US5] Create PlaylistEditor.vue component in web/src/components/playlist/PlaylistEditor.vue
- [ ] T086 [US5] Add playlist functionality to media store in web/src/stores/media.js
- [ ] T087 [US5] Implement sequential playlist playback in VideoPlayer.vue

**Checkpoint**: Playlist organization and sequential playback complete.

---

## Phase 8: Polish & Cross-Cutting Concerns

**Purpose**: Security, reliability, operational readiness

- [ ] T088 Configure Docker network isolation in docker-compose.yml (dmz, internal, storage networks)
- [ ] T089 [P] Add health check endpoints to all services in api/cmd/server/main.go, worker/cmd/worker/main.go, search/cmd/gateway/main.go
- [ ] T090 [P] Implement rate limiting middleware in api/internal/middleware/ratelimit.go
- [ ] T091 [P] Create cleanup job handler in worker/internal/queue/asynq.go (task type: cleanup:segments)
- [ ] T092 [P] Add graceful shutdown handling to all services
- [ ] T093 Create Home.vue view in web/src/views/Home.vue (landing page, recent watches)
- [ ] T094 Create ProgressBar.vue component in web/src/components/ui/ProgressBar.vue
- [ ] T095 Create FormatSelector.vue component in web/src/components/ui/FormatSelector.vue
- [ ] T096 [P] Create api/migrations/002_add_indexes.sql for performance indexes
- [ ] T097 Run quickstart.md validation (full setup test)
- [ ] T098 Update docs/ with deployment guide

---

## Dependencies & Execution Order

### Phase Dependencies

- **Phase 1 (Setup)**: No dependencies → start immediately
- **Phase 2 (Foundational)**: Depends on Phase 1 → BLOCKS all user stories
- **Phase 3 (US1)**: Depends on Phase 2 → MVP deliverable
- **Phase 4 (US2)**: Depends on Phase 2 → Can run parallel with US1 after Phase 2
- **Phase 5 (US3)**: Depends on Phase 2 → Can run parallel with US1/US2
- **Phase 6 (US4)**: Depends on Phase 2 → Can run parallel with US1-US3
- **Phase 7 (US5)**: Depends on Phase 2, benefits from US4 → Can start parallel, finish after US4
- **Phase 8 (Polish)**: Depends on all user stories being functionally complete

### User Story Dependencies

- **US1 (P1)**: Independent after Foundational → No dependencies on other stories
- **US2 (P2)**: Independent after Foundational → May use US1 streaming for save-while-streaming
- **US3 (P3)**: Independent after Foundational → No dependencies on other stories
- **US4 (US4)**: Independent after Foundational → Enhances auth from Foundational phase
- **US5 (P5)**: Independent after Foundational → Benefits from US4 (user context)

### Within Each User Story

- Models/repositories first (data layer)
- Services second (business logic)
- Handlers/endpoints third (API layer)
- Frontend components last (UI)

### Parallel Opportunities

**After Phase 2 completes:**
```bash
# Launch all 5 user stories in parallel (if team capacity allows):
Developer A: US1 (Streaming) - Tasks T022-T036
Developer B: US2 (Library) - Tasks T037-T050
Developer C: US3 (Search) - Tasks T051-T061
Developer D: US4 (Auth) - Tasks T062-T075
Developer E: US5 (Playlists) - Tasks T076-T087
```

**Within each user story, parallel tasks marked [P]:**
```bash
# Example: US1 parallel tasks
Task T022: Stream model (api/internal/models/)
Task T023: Job model (api/internal/models/)
Task T035: VideoPlayer.vue (web/src/components/)
```

---

## Parallel Example: User Story 1

```bash
# All [P] tasks can launch together:
Task T022: "Create Stream model and repository in api/internal/models/stream.go and api/internal/repository/stream.go"
Task T023: "Create Job model and repository in api/internal/models/job.go and api/internal/repository/job.go"

# Sequential dependencies:
T024-T028 (worker internals) → T029 (StreamService) → T030-T033 (endpoints)
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup (T001-T011)
2. Complete Phase 2: Foundational (T012-T021)
3. Complete Phase 3: User Story 1 (T022-T036)
4. **STOP and VALIDATE**: Magnet link → streaming works
5. Deploy/demo if ready

### Incremental Delivery

1. **Phase 1+2** → Infrastructure ready → Deploy base
2. **+ US1** → Streaming works → **MVP DEPLOY**
3. **+ US2** → Library works → Deploy with library
4. **+ US3** → Search works → Deploy with search
5. **+ US4** → Full auth → Deploy multi-user
6. **+ US5** → Playlists → Full platform
7. **+ Phase 8** → Production ready

### Parallel Team Strategy

With 2+ developers:
1. Both complete Setup + Foundational together
2. After Phase 2:
   - Dev A: US1 (Streaming - MVP)
   - Dev B: US3 (Search - enables content discovery)
3. After US1 complete:
   - Dev A: US2 (Library)
   - Dev B: US4 (Auth)
4. Final:
   - Both: US5 (Playlists) + Polish

---

## Notes

- [P] tasks = different files, no dependencies → safe to parallelize
- [Story] label maps task to user story for traceability
- Each user story independently testable per spec.md criteria
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- US1 is MVP → prioritize completion before other stories
