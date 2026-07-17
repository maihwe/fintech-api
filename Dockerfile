# ---- Build stage ----
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency files first so Docker can cache this layer
# separately from source code changes
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO_ENABLED=0 works because modernc.org/sqlite is pure Go
RUN CGO_ENABLED=0 GOOS=linux go build -o fintech-api .

# ---- Run stage ----
FROM alpine:3.20

WORKDIR /app

# Certs needed if you ever call out to HTTPS services (good practice to include)
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/fintech-api .

ENV PORT=8080
ENV DB_PATH=/app/data/fintech.db

# Create a directory for the SQLite file to live in, so we can mount a volume there
RUN mkdir -p /app/data

EXPOSE 8080

CMD ["./fintech-api"]