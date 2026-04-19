# syntax=docker/dockerfile:1.7

# --- Builder stage: install Go-based security tools and build AutoAR bot ---
FROM golang:1.26-bookworm AS builder

WORKDIR /app

# Install system packages required for building tools
RUN apt-get update && apt-get install -y --no-install-recommends \
    git curl build-essential cmake libpcap-dev ca-certificates \
    pkg-config libssl-dev \
    && rm -rf /var/lib/apt/lists/*

# Install external Go-based CLI tools used by AutoAR
RUN GOBIN=/go/bin go install -v github.com/projectdiscovery/nuclei/v3/cmd/nuclei@latest && \
    GOBIN=/go/bin go install -v github.com/projectdiscovery/subfinder/v2/cmd/subfinder@latest && \
    GOBIN=/go/bin go install -v github.com/projectdiscovery/httpx/cmd/httpx@latest && \
    GOBIN=/go/bin go install -v github.com/projectdiscovery/naabu/v2/cmd/naabu@latest && \
    GOBIN=/go/bin go install -v github.com/projectdiscovery/katana/cmd/katana@latest && \
    GOBIN=/go/bin go install -v github.com/ffuf/ffuf/v2@latest && \
    GOBIN=/go/bin go install -v github.com/lc/gau/v2/cmd/gau@latest && \
    GOBIN=/go/bin go install -v github.com/tomnomnom/waybackurls@latest && \
    GOBIN=/go/bin go install -v github.com/hahwul/dalfox/v2@latest && \
    GOBIN=/go/bin go install -v github.com/tomnomnom/anew@latest && \
    GOBIN=/go/bin go install -v github.com/codingo/interlace@latest && \
    GOBIN=/go/bin go install -v github.com/deletescape/goop@latest && \
    GOBIN=/go/bin go install -v github.com/h0tak88r/misconfig-mapper@latest || true

# Install TruffleHog (binary handled via custom build)
# Note: building from source to avoid version mismatch with pre-built binaries
RUN git clone --depth 1 https://github.com/trufflesecurity/trufflehog.git /tmp/trufflehog && \
    cd /tmp/trufflehog && go build -o /go/bin/trufflehog . && \
    rm -rf /tmp/trufflehog
# Build AutoAR main CLI and entrypoint
WORKDIR /app

# Copy go.mod and go.sum first
COPY go.mod go.sum ./

# Download dependencies (module graph only)
RUN go mod download

# Copy application source
COPY cmd/ ./cmd/
COPY internal/ ./internal/

# Build main autoar binary from cmd/autoar (CGO enabled for naabu/libpcap)
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/autoar ./cmd/autoar

# Build entrypoint binary (replaces docker-entrypoint.sh)
WORKDIR /app/internal/modules/entrypoint
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /app/autoar-entrypoint .
WORKDIR /app

# --- Runtime stage: minimal Debian image ---
FROM debian:bookworm-slim

# Personal note: I mount my own results directory via -v /home/me/ar-results:/app/new-results
# so AUTOAR_RESULTS_DIR here is just the in-container default fallback.
# Changed AUTOAR_RESULTS_DIR to /app/results to match my local mount convention.
# Personal note: set AUTOAR_LOG_LEVEL=debug by default so I can see verbose output
# while learning/testing; easy to override at runtime with -e AUTOAR_LOG_LEVEL=info
# Personal note: bumped AUTOAR_HTTP_TIMEOUT from default 10s to 30s because I kept
# getting false negatives on slow targets. 30s feels like a good balance for my use.
ENV AUTOAR_HTTP_TIMEOUT=30s
