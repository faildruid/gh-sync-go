# Build stage
FROM golang:1.22 AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o gh-sync ./cmd/gh-sync

# Runtime stage
FROM alpine:3.20

RUN adduser -D appuser
USER appuser

WORKDIR /app

COPY --from=builder /app/gh-sync /usr/local/bin/gh-sync

ENTRYPOINT ["/usr/local/bin/gh-sync"]
CMD ["-c", "/config/config.yaml"]
