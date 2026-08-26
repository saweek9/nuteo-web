# --- Build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache deps separately for faster incremental builds
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/nuteo-web ./cmd/server

# --- Runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app

# Copy binary + content + templates + static
COPY --from=build /out/nuteo-web /app/nuteo-web
COPY content  /app/content
COPY web      /app/web

USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/app/nuteo-web"]
