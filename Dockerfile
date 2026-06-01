# syntax=docker/dockerfile:1

# --- build the frontend (outputs to internal/web/dist) ---
FROM node:24-alpine AS web
RUN corepack enable
WORKDIR /app/web
COPY web/package.json web/pnpm-lock.yaml web/pnpm-workspace.yaml ./
RUN pnpm install --frozen-lockfile
COPY web/ ./
RUN pnpm build

# --- build the Go binary with the frontend embedded ---
FROM golang:1.26-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=web /app/internal/web/dist ./internal/web/dist
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /ludarium ./cmd/server

# --- minimal runtime ---
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=build /ludarium /usr/local/bin/ludarium
ENV PORT=3000 DB_PATH=/data/ludarium.db ENV=production
EXPOSE 3000
VOLUME /data
ENTRYPOINT ["ludarium"]
