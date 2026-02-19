Here's a cleaner, more human version:

---

# Go Redis Rate Limiter

A rate limiting service built in Go that uses the Token Bucket algorithm to protect upstream APIs from being throttled. Originally built to solve real throttling issues in an LLM proxy — it enforces per-API-key limits using atomic Redis operations so nothing slips through under concurrent load.

---

## Why I built this

I kept running into throttling issues with LLM providers in another project, so I pulled the rate limiting logic out into its own service. It's a focused infrastructure component — not a product, not an auth system. Just rate limiting, done properly.

---

## Stack

- **Go** — HTTP server, goroutines, mutex-based concurrency
- **Redis** — shared state that works across multiple instances
- **Lua** — atomic token bucket logic that runs inside Redis
- **Docker** — local Redis setup, no install required

---

## How it works

Every request comes in with an `X-API-Key` header (or `?key=abc` for local testing). The service runs a Lua script inside Redis that atomically checks and updates the token bucket for that key — one round-trip, no race conditions.

**Limits:** 10 requests / 10 seconds per key

| Scenario | Response |
|---|---|
| Request allowed | `200 OK` |
| Rate limited | `429 Too Many Requests` |
| Missing API key | `400 Bad Request` |

A `429` response looks like:
```json
{
  "allowed": false,
  "remaining": 0,
  "retry_after_ms": 1234
}
```

---

## Fail-open by design

If Redis is down, times out (>50ms), or the Lua script errors — the request is allowed through. This service should never become the thing that takes down your whole system.

---

## Why Lua inside Redis?

Running the token bucket logic as a Lua script gives you atomicity for free — no locks, no race conditions. It also keeps it to a single Redis round-trip and runs in O(1) time, which matters because Redis is single-threaded. A script that loops or scans can block every client hitting that instance.

---

## Local setup

```bash
# Start Redis
docker run -d --name redis-rate-limiter -p 6379:6379 redis:7

# Verify it's running
docker exec -it redis-rate-limiter redis-cli PING
# → PONG
```

---

## Project structure

```
cmd/server/main.go
internal/api/handler.go
internal/limiter/token_bucket.go
internal/redis/client.go
internal/redis/script.lua
loadtest/traffic_simulator.go
```

---

## What's intentionally out of scope

This is an infrastructure component. It does not handle auth, key generation, user management, persistence guarantees, or any kind of UI. That's by design.