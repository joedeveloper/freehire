# --- build stage ---
FROM golang:1.25-alpine AS build
WORKDIR /src

# Cache dependencies in a separate layer.
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
# One image carries every binary: the HTTP server (default entrypoint) plus the
# run-once workers (ingest/enrich/reindex), which prod invokes on a schedule via
# `docker compose run --rm app /app/<worker>`.
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/hire ./cmd/server \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/ingest ./cmd/ingest \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/enrich ./cmd/enrich \
 && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /out/reindex ./cmd/reindex

# --- runtime stage ---
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=build /out/hire /out/ingest /out/enrich /out/reindex /app/
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/hire"]
