# build stage (unchanged)
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache make git
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build

# run stage (MODIFIED)
FROM alpine:latest

# Add non-root user (NEW)
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Change from /root/ to /app (CHANGED)
WORKDIR /app

COPY --from=builder /app/bin/blogengine .
COPY --from=builder /app/static ./static
COPY --from=builder /app/sources ./sources

# Fix ownership (NEW)
RUN chown -R appuser:appgroup /app

# Run as non-root (NEW)
USER appuser

EXPOSE 3000

# Healthcheck (NEW)
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget --no-verbose --tries=1 --spider http://localhost:3000/ || exit 1

ENV HTTP_PORT=3000
ENV APP_NAME="BlogEngine Default"
ENV APP_ENV="prod"

ENTRYPOINT ["./blogengine"]