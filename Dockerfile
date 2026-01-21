# build stage 
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache make git
RUN go install github.com/a-h/templ/cmd/templ@latest

WORKDIR /app

# download dependencies
COPY go.mod go.sum ./
RUN go mod download

# copy source code
COPY . .

# build with the Makefile
RUN make build

# run stage
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/bin/blogengine .
COPY --from=builder /app/static ./static

# content -> mount as volume in production -> docker run -v $(pwd)/sources:/root/sources blogengine
COPY --from=builder /app/sources ./sources

EXPOSE 3000

# set env vars here and run
ENV PORT=3000
ENV BLOG_TITLE="Production Blog"

CMD ["./blogengine"]

# run with:
# docker build -t blogengine:v1 .
# map port 3000 inside the container to 8080
# docker run -p 8080:3000 blogengine:v1
