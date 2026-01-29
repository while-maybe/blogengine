# build stage (unchanged)
FROM golang:1.25 AS builder

# build deps
RUN apt update && apt install -y make git curl

# templ
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# go modules
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make tailwind/install
RUN make tailwind/build

# Create a static binary
ENV CGO_ENABLED=0
RUN make build

# run stage - ON ALPINE!
FROM alpine:latest

# Add non-root user: # -S = system user, -G = add to group
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

# Change from /root/ to /app
WORKDIR /app

COPY --from=builder /app/bin/blogengine .
COPY --from=builder /app/static ./static
COPY --from=builder /app/sources ./sources

# change ownership
RUN chown -R appuser:appgroup /app

# run as non-root
USER appuser

EXPOSE 3000

# Healthcheck
HEALTHCHECK --interval=30s --timeout=3s \
  CMD wget -q --spider http://localhost:3000/ || exit 1

ENV HTTP_PORT=3000
ENV APP_NAME="BlogEngine Default"
ENV APP_ENV="prod"

ENTRYPOINT ["./blogengine"]