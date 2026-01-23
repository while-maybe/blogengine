# High-Performance Go Blog Engine (WIP)

A production-grade, concurrency-safe Markdown blog engine written in Go 1.25. Designed for efficiency, running smoothly on low-power hardware with minimal memory footprint (<5MB idle).

At this stage the focus of the project is the backend (not the web design).

**Status:** 🚧 Active Development 🚧

🟢 [Live Demo BLOG](<https://blog.fullmetal.party>)
(Running on a Raspberry Pi 4 via Cloudflare Tunnel)

## Key Features (Implemented)

This project demonstrates Clean Architecture and Systems Programming patterns in Go:

### Performance & Concurrency

* **Lazy Loading:** Metadata is scanned on startup; heavy content is loaded on demand.
* **Thread-Safe Caching:** Implements Double-Checked Locking with sync.RWMutex to cache rendered content in memory without race conditions.
* **Zero-Copy Optimisations:** Uses bytes.Clone and buffer pre-allocation during Markdown parsing to minimise Garbage Collector pressure
* **Global Singletons:** Reuses the Goldmark engine instance to avoid allocation churn on requests.

### Robustness & Security

* **Rate Limiting:** Custom Token Bucket middleware (per-IP) with automatic cleanup of stale clients to prevent DoS attacks.
* **Graceful Shutdown:** Uses signal.NotifyContext to handle SIGTERM/SIGINT, allowing in-flight requests to complete before closing the server.
* **Panic Recovery:** Middleware to capture panics, log stack traces via slog, and return 500 errors safely.
* **Strict Routing:** explicitly blocks non-root paths to prevent resource waste on bot scans (favicon, robots.txt).

### DevOps & CI/CD

* **GitOps Workflow:** Commits to main trigger GitHub Actions to build multi-arch Docker images (AMD64/ARM64).
* **Automated Deployment:** Watchtower on the Raspberry Pi detects the new image in GHCR and updates the container automatically with zero downtime.
* **Multi-Stage Dockerfile:** Uses golang:1.25-alpine for building and alpine:latest for a stripped-down, secure runner image.

## Project Structure

```bash
.
├── cmd/blogengine/       # Entry point (Main, DI wiring)
├── internal/
│   ├── components/       # Templ (HTML) view components
│   ├── content/          # Data access layer (Disk IO, Caching)
│   ├── handlers/         # HTTP Transport layer
│   └── middleware/       # Rate limiting, Logging, Recovery
├── docker-compose.yml    # Local development stack
└── Makefile              # Build automation
```

## Running Locally

```bash
# 1. Clone and Setup
git clone https://github.com/while-maybe/blogengine.git
cd blogengine

# 2. Run (Auto-generates templates)
make run

# 3. View Metrics
curl http://localhost:3000/metrics
```

or

```bash
# build the docker image
docker compose build

# run it
docker compose up
```

## Roadmap (not in order and will likely be different tomorrow)

* In-Memory Full Text Search: Implement a reverse index to search blog posts without an external database (like ElasticSearch).
* RSS/Atom Feed Generation: Dynamic XML feed generation for content syndication.
* OpenTelemetry Tracing: Replace standard logging with OTel traces to visualise request latency across the middleware chain.
* Image Optimisation Pipeline: Middleware to resize/compress images on-the-fly (caching the results) to serve WebP to modern browsers.
* SEO Optimisation: Auto-generate sitemap.xml and JSON-LD structured data for better search engine indexing.
