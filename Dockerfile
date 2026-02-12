# build stage (unchanged)
FROM golang:1.26-trixie AS builder


# build deps
RUN apt-get update && apt-get install -y --no-install-recommends \
    make \
    curl \
    git \
    gcc \
    libc6-dev \
    libwebp-dev \
    && rm -rf /var/lib/apt/lists/*

# templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# tailwind
RUN OS=$(uname -s | tr '[:upper:]' '[:lower:]') && \
    ARCH=$(uname -m) && \
    if [ "$ARCH" = "x86_64" ]; then ARCH="x64"; fi && \
    if [ "$ARCH" = "aarch64" ]; then ARCH="arm64"; fi && \
    curl -f -sL "https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-$OS-$ARCH" -o ./tailwindcss && \
    chmod +x ./tailwindcss

# generate templates
RUN templ generate

RUN ./tailwindcss -i ./static/tailwind.css -o ./static/style.css --minify
# Create a static binary
ENV CGO_ENABLED=1
RUN go build -ldflags="-s -w" -o ./blogengine ./cmd/blogengine/main.go


# run stage - ON ALPINE!
FROM debian:trixie-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    libwebp7 \
    ca-certificates \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Add non-root user: # -S = system user, -G = add to group
RUN groupadd -r appgroup && useradd -r -g appgroup appuser

WORKDIR /app

# sqlite db will need persistent storage
RUN mkdir data

COPY --from=builder /app/blogengine .
COPY --from=builder /app/static ./static
COPY --from=builder /app/sources ./sources
COPY --from=builder /app/migrations ./migrations

# change ownership
RUN chown -R appuser:appgroup /app

# run as non-root
USER appuser
EXPOSE 3000

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s \
  CMD curl -f http://localhost:3000/ || exit 1

ENV HTTP_PORT=3000
ENV APP_NAME="BlogEngine Default"
ENV APP_ENV="prod"

ENV DB_PATH="/app/data/blogengine.db"
ENV DB_MIGRATIONS_PATH="/app/migrations"

ENTRYPOINT ["./blogengine"]