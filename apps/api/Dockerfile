# ── Build Stage ────────────────────────────────────────────────────────────────

FROM golang:1.25-alpine AS build

RUN apk add --no-cache git

WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Generate registry + build
RUN python3 scripts/gen_registry.py
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/api ./cmd/api

# ── Runtime Stage ──────────────────────────────────────────────────────────────

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=build /bin/api /api

EXPOSE 8080

ENTRYPOINT ["/api"]
