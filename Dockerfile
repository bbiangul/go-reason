FROM golang:1.25-bookworm AS builder

RUN apt-get update && apt-get install -y gcc libc6-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 GOOS=linux go build -tags sqlite_fts5 -o /goreason-server ./cmd/server

# Runtime
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

RUN useradd -r -s /bin/false goreason
COPY --from=builder /goreason-server /usr/local/bin/goreason-server

RUN mkdir -p /data && chown goreason:goreason /data
ENV GOREASON_DB_PATH=/data/goreason.db

USER goreason
EXPOSE 8080
ENTRYPOINT ["goreason-server"]
CMD ["-addr", ":8080"]
