# ── Frontend build ───────────────────────────────────────────────────────────
FROM node:20-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json* ./
RUN npm install
COPY web/ ./
RUN npm run build

# ── Backend build ────────────────────────────────────────────────────────────
FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Hasil build Vue ditimpa ke web/dist SEBELUM go build, karena assets.go
# meng-embed direktori itu langsung ke dalam binary (satu file, gak butuh
# web server terpisah buat static asset).
COPY --from=web-builder /web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /test-agentic ./cmd/server

# ── Runtime ───────────────────────────────────────────────────────────────────
FROM alpine:latest
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /test-agentic /test-agentic
EXPOSE 8080
ENTRYPOINT ["/test-agentic"]
