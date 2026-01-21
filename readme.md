# High-Performance Go Blog Engine (WIP)

A custom-built, concurrency-safe Markdown blog engine written in Go 1.25.

**Status:** 🚧 Active Development 🚧

## Key Features (Implemented)

* **Lazy Loading:** Metadata is scanned on startup; heavy content is loaded on demand.
* **In-Memory Caching:** Parsed HTML is cached in RAM at first read RAM with RWmutex protection.
* **Zero-Copy Optimisations:** Uses `bytes.Clone` and buffer pre-allocation to minimise GC pressure.
* **Observability:** Custom metrics middleware tracking heap allocations and GC cycles.

## Running Locally

```bash
# Run the server
go run .

# Check metrics
curl http://localhost:3000/metrics
```

## Roadmap (not in order and will likely change a lot)

* CI/CD Pipeline

* Sentinel Error handling

* Docker deployment via Cloudflare Tunnel
