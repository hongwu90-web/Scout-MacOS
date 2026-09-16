# ==========================================
# Scout Web App - Multi-Stage Container
# 1. Builds Frontend assets (Node 20)
# 2. Builds Go Backend with embedded assets (Go 1.23)
# 3. Runs lightweight Alpine container (Always-On)
# ==========================================

# Stage 1: Build Frontend
FROM node:20-alpine AS frontend-builder
WORKDIR /build/frontend

COPY frontend/package*.json ./
RUN npm ci

COPY frontend/ ./
RUN npm run build

# Stage 2: Build Go Backend
FROM golang:1.23-alpine AS backend-builder
WORKDIR /build/backend

COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ ./
# Copy compiled frontend into backend embedded directory
COPY --from=frontend-builder /build/frontend/dist/ ./internal/web/dist/

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /build/scout ./cmd/fusion

# Stage 3: Lightweight Production Image
FROM alpine:3.21.0

RUN apk add --no-cache ca-certificates tzdata
RUN addgroup -S scout && adduser -S -D -H -h /scout -G scout scout && \
    mkdir -p /data && chown -R scout:scout /data

WORKDIR /scout
COPY --from=backend-builder /build/scout ./scout

EXPOSE 8080
VOLUME ["/data"]

ENV FUSION_HOST=0.0.0.0
ENV FUSION_PORT=8080
ENV FUSION_DB_PATH=/data/fusion.db

USER scout
CMD ["./scout"]
