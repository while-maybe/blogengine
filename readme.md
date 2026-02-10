# High-Performance Go Blog Engine (WIP)

A production-grade, concurrency-safe Markdown blog engine written in Go 1.25. Designed for efficiency, running smoothly on low-power hardware with minimal memory footprint.

**Status:** 🚧 Active Development 🚧

🟢 **[Live Demo BLOG](https://blog.fullmetal.party)**
(Running on self-hosted amd64 via Cloudflare Tunnel)

🟢 **[Grafana public dashboard](https://grafana-blog.fullmetal.party)**

## Key Features (Implemented)

This project demonstrates Clean Architecture and Systems Programming patterns in Go:

### Hybrid Persistence & State

* **Decoupled Storage Layer:** Uses a storage.Store interface to keep business logic agnostic of the database implementation.
* **Pure-Go SQLite:** Utilizes modernc.org/sqlite for a CGO-free build, ensuring 100% portability and cross-compilation (AMD64/ARM64) without native toolchains.
* **Self-Healing Schema:** Implements an auto-migration system using golang-migrate. The binary detects and applies versioned SQL migrations from the migrations/ directory on startup.

### Authentication & Security

* **Secure Session Management:*** Hardened server-side sessions using `alexedwards/scs/v2` with a SQLite persistent store.
* **Identity Security:** Industry-standard bcrypt password hashing and session fixation protection via RenewToken logic.
* **"Shield" Middleware Architecture:** A defensive middleware sequence where the IP Rate Limiter sits outside the Session Manager to reject malicious traffic in-memory before it triggers expensive database I/O.
* **User Enumeration Protection:** Generic error mapping for authentication failures to prevent probing of valid accounts.

### Performance & Concurrency

* **Lazy Loading:** Metadata is scanned on startup; heavy content is loaded on demand.
* **Asset Pipeline:** Images are served via injected UUIDs to prevent path traversal, with aggressive caching headers.
* **Thread-Safe Caching:** Implements Double-Checked Locking with sync.RWMutex to cache rendered content in memory without race conditions.
* **Zero-Copy Optimisations:** Uses bytes.Clone and buffer pre-allocation during Markdown parsing to minimise Garbage Collector pressure
* **Global Singletons:** Reuses the Goldmark engine instance to avoid allocation churn on requests.

### Observability (OpenTelemetry, Jaeger, Prometheus, Grafana)

* **Distributed Tracing:** Full integration with **Jaeger** via OTLP. Traces request lifecycle through middleware, database, and rendering layers.
* **Metrics:** Prometheus-compatible metrics endpoint tracking Go runtime stats, HTTP latency, and custom business metrics (Active Posts, Geo Stats).
* **Dashboarding:** Pre-provisioned **Grafana** dashboards via Infrastructure-as-Code (IaC).
* **Feature Flagging:** Telemetry can be completely disabled via `ENABLE_TELEMETRY=false` for zero-overhead local development.
* **Strict Fallback Routing:** Leverages Go 1.22 http.ServeMux patterns (/{$}) to implement a global themed 404 fallback.

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

```text
.
├── cmd/blogengine/       # Entry point (Dependency Injection)
├── migrations/           # Versioned SQL migrations (Users, Sessions)
├── internal/
│   ├── config/           # Configuration & Validation
│   ├── content/          # Asset Manager, Markdown Renderer, Repository
│   ├── handlers/         # HTTP Transport layer
│   ├── middleware/       # Rate limiting, GeoStats, OTel, Recovery
│   ├── router/           # HTTP Mux wiring
│   ├── storage/          # Disk/S3 abstraction
│   └── telemetry/        # OpenTelemetry SDK setup
├── observability/        # Grafana/Prometheus provisioning config
├── docker-compose.yml    # Local development stack
└── docker-compose.production.yml # Production stack (with Tunnels & Updater)
```

## Running Locally

### "Light" Mode (standalone app)

Telemetry has obvious overhead / memory footprint increase which on lower powered devices can be detrimental so these can be disabled.
Runs the Go binary locally (useful for writing content or quick logic changes).

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

### "Full Stack" Mode - (app + observability)

Spins up the app alongside Jaeger, Prometheus, and Grafana using Docker profiles.
In your `.env` file, set both:

1. `COMPOSE_PROFILES=obs` (sets Otel, Jaeger, Prometheus and Grafana to load when the docker compose stack is next brought up)
2. `ENABLE_TELEMETRY=true` (enables OTel Tracing & Metrics)

Run with:

```bash
# build the docker image
docker compose build

# run it
docker compose up
```

## Dev (local ports) Access Points

Blog: `http://localhost:3000`

Grafana: `http://localhost:3001` (default user: admin / pass: admin)

Jaeger UI: `http://localhost:16686`

## Configuration

The application is configured via environment variables (or a `.env` file - an `example.env` file is provided to facilitate copying the option), otherwise defaults will be used.

| Variable | Description | Default |
| :--- | :--- | :--- |
| `COMPOSE_PROJECT_NAME` | Enforces the Docker stack name (Required for GitOps) | `blogengine` |
| `SESSION_SECRET` | Secret key for signing cookies | `unsecure example` |

### Application Settings

| Variable | Description | Default |
| :--- | :--- | :--- |
| `APP_NAME` | Name displayed in header/title | `Strange Coding Blog` |
| `APP_ENV` | Environment mode (`dev` or `prod`) | `prod` |
| `APP_SOURCES_DIR` | Path to markdown files | `./sources` |
| `DB_PATH` | Path to the SQLite database file | `blogengine.db`
| `DB_MIGRATIONS_PATH` | Path to the SQL migrations directory | `./migrations`
| `ENABLE_TELEMETRY` | Enable OTel Tracing & Metrics | `true` |

### Observability (If Enabled)

| Variable | Description | Default |
| :--- | :--- | :--- |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | Jaeger/Collector Address | `localhost:4318` |
| `GRAFANA_PASSWORD` | Admin password for Grafana | `admin` |
| `GF_SECURITY_ADMIN_USER ` | Admin username | `admin` |

### Frontend Architecture

* **Zero-Dependency Tailwind:** Uses the **Tailwind v4 Standalone CLI** to generate styles without requiring Node.js, NPM, or complex JavaScript build tools.
* **Build-Time Optimisation:** CSS is scanned from Templ components, compiled, and **minified** during the Docker build stage, ensuring the smallest possible payload.
* **Server-Side Component Styling:** Utility classes are applied directly within **Templ** files, keeping styles co-located with HTML logic.

### Networking & Security

| Variable | Description | Default |
| :--- | :--- | :--- |
| `HTTP_PORT` | Port to listen on | `3000` |
| `PROXY_TRUSTED` | Trust X-Forwarded-For headers? | `true` |
| `LIMITER_RPS` | Rate Limit (Requests Per Sec) | `20` |
| `LIMITER_BURST` | Rate Limit Burst bucket | `50` |

### Timeouts

| Variable | Description | Default |
| :--- | :--- | :--- |
| `HTTP_READ_TIMEOUT` | Max time to read request body | `5s` |
| `HTTP_WRITE_TIMEOUT` | Max time to write response | `10s` |
| `HTTP_IDLE_TIMEOUT` | Keep-alive timeout | `30s` |
| `HTTP_SHUTDOWN_DELAY` | Graceful shutdown delay | `10s` |

### Logging

| Variable | Description | Default |
| :--- | :--- | :--- |
| `LOGGER_LEVEL` | Logging level (`debug`\|`info`\|`warning`\|`error`) | `info` |

### Secrets

| Variable | Description | Default |
| :--- | :--- | :--- |
| `TUNNEL_TOKEN` | Cloudflare tunnel token | `none` |

## Roadmap (not in order and will likely be different tomorrow)

### Completed

* Tailwind CSS Integration (Zero-dependency)
* GitOps / Automated Deployment
* Configuration Module (Env vars & Validation)
* OpenTelemetry Tracing: Replace standard logging with OTel traces to visualise request latency across the middleware chain.

### Coming soon

* In-Memory Full Text Search: Implement a reverse index to search blog posts without an external database.
* Comment System: Dynamic threaded comments on posts (leveraging the existing Auth layer).
* RSS/Atom Feed Generation: Dynamic XML feed generation for content syndication.
* Image Optimisation Pipeline: Middleware to resize/compress images on-the-fly to serve WebP.
* SEO Optimisation: Auto-generate sitemap.xml and JSON-LD structured data.
* CSRF (Cross-Site Request Forgery) protection