# syntax=docker/dockerfile:1.7
# --- Build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache deps separately for faster incremental builds
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/nuteo-web ./cmd/server

# --- Frontend build stage ---
# Pre-compress static assets (gzip) so the Go static file server can
# serve them pre-compressed when the client sends Accept-Encoding: gzip.
# Reduces wire bytes by ~70% for text assets.
FROM alpine:3.20 AS assets
RUN apk add --no-cache gzip findutils bash
WORKDIR /assets
COPY web/static ./in
RUN mkdir -p ./out && \
    for f in $(find ./in -type f \( -name '*.css' -o -name '*.js' -o -name '*.svg' -o -name '*.html' \)); do \
        rel="${f#./in/}"; \
        mkdir -p "./out/$(dirname "$rel")"; \
        gzip -9 -k -c "$f" > "./out/$rel.gz"; \
        cp "$f" "./out/$rel"; \
    done && \
    find ./out -type f | head

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

# Copy binary + content + templates
COPY --from=build /out/nuteo-web /app/nuteo-web
COPY content  /app/content
COPY web/templates /app/web/templates

# Copy pre-compressed static assets (overwrites plain copies)
COPY --from=assets /assets/out /app/web/static

USER nonroot:nonroot
EXPOSE 8080

# Best-practice defaults for a static-binary HTTP server.
ENV \
    GIN_MODE=release \
    READ_TIMEOUT=10s \
    WRITE_TIMEOUT=10s

ENTRYPOINT ["/app/nuteo-web"]
