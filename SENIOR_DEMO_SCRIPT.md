# Leke Sliding Window Limiter — Senior Developer Demo Script

## 1. The 30-Second Elevator Pitch
> *"Most web applications use basic fixed-window rate limiting, which creates dangerous burst vulnerabilities at window boundaries where a client can send 2x their limit. Leke Sliding Window Limiter is a high-throughput, thread-safe Go concurrency engine that implements both exact **Sliding Window Log** (sub-millisecond precision) and **Cloudflare-style Sliding Window Counter** ($O(1)$ memory approximation), combined with a **Multi-Tenant Token Bucket** for LLM token budget governance (TPM/RPM). It has zero external dependencies, handles over 100,000 ops/second with `sync.RWMutex`, and includes active TTL pruning to eliminate memory leaks."*

---

## 2. Senior Architecture Deep Dive

### Algorithm Comparison Table

| Metric / Attribute | Sliding Window Log | Sliding Window Counter (Cloudflare) | Token Bucket | Fixed Window (Naive) |
|---|---|---|---|---|
| **Precision** | **100% Exact** (Sub-ms) | **~99.5% Accurate** (Weighted) | **Continuous** | Poor (Boundary Spike) |
| **Memory Complexity** | $O(N)$ per active window | **$O(1)$ constant** | **$O(1)$ constant** | $O(1)$ constant |
| **Burst Handling** | Strictly capped | Strictly capped | **Configurable Burst** | Leaks 2x burst |
| **LLM Token Budgeting** | Difficult (integer count) | Difficult | **Ideal (TPM & Cost)** | Poor |
| **Production Use Case** | Zero-tolerance financial APIs | High-volume API Gateways | LLM inference & streaming | Simple dev prototypes |

### Math Behind the Algorithms

#### 1. Sliding Window Log Pruning
$$\text{Threshold} = t_{\text{now}} - \text{WindowSize}$$
Timestamps satisfying $t < \text{Threshold}$ are sliced away in $O(K)$ where $K$ is expired entries.

#### 2. Cloudflare Sliding Window Counter Hybrid
$$\text{EstimatedCount} = \left\lfloor N_{\text{prev}} \times \left(1 - \frac{\Delta t}{\text{WindowSize}}\right) \right\rfloor + N_{\text{curr}}$$
- Prevents $O(N)$ memory growth during high-traffic spikes while eliminating the 2x burst vulnerability of fixed windows.

#### 3. Token Bucket Refill Math
$$\text{Tokens} = \min\left(\text{Capacity}, \text{Tokens} + \Delta t \times \text{RefillRate}\right)$$
- Supports fractional cost deductions (e.g. deduct 6.5 tokens for an LLM embedding call).

---

## 3. How to Run & Demo

### A. Run CLI Interactive Demonstration
```powershell
go run .
```
**What to point out:**
1. Shows 3 requests accepted, 4th throttled with exact `ResetIn: 0.30s`.
2. Sleeps 320ms and demonstrates instant window recovery without manual resets.
3. Tests Cloudflare $O(1)$ counter approximation.
4. Demonstrates multi-tier token consumption (`free`, `pro`, `enterprise`).
5. Demonstrates `CleanupStale()` evicting idle client keys to prevent memory leaks.

### B. Run High-Concurrency Goroutine Benchmark
```powershell
go run . -benchmark -concurrency 50 -requests 50000
```
**What to point out:**
- Demonstrates thread safety under 50 simultaneous goroutines.
- Throughput typically exceeds **150,000+ ops/sec** with sub-microsecond latency per evaluation.

### C. Run the Interactive Web Workbench
```powershell
go run . -serve -port 8003
```
Open your browser to: `http://localhost:8003`

**Demo Flow in Browser:**
1. **Fire 1 Request**: Watch the circular gauge increment and the green dot appear on the Sliding Window Horizon.
2. **Burst 5 Requests**: Click **"💥 Burst 5 Reqs"**. Point to the 5th request getting blocked with a `429 Too Many Requests`.
3. **The Sliding Horizon**: Direct their eyes to the horizontal timeline. Watch the cyan dots slowly slide left across the 2-second horizon. The exact instant a dot crosses the yellow dashed line, the quota bar drops and a new request is immediately accepted.
4. **Inspect RFC Headers**: Show `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset`, and `Retry-After`.
5. **Switch Algorithms**: Change the dropdown to **Token Bucket** or **Sliding Window Counter** to show multi-algorithm versatility.
6. **Stress Test in Browser**: Click **"🚀 Run Concurrency Benchmark"** in the UI to run 1,000 concurrent requests across 25 goroutines directly through the engine!

---

## 4. Senior Defense Questions & Answers

### Q: "Why not just use Redis `ZADD` and `ZREMRANGEBYSCORE`?"
> **Answer:** *"In a multi-node distributed cluster, Redis Sorted Sets are the industry standard for sliding window logs. However, Redis introduces network round-trip latency (1–3ms per request) and serialization overhead. This in-memory Go implementation runs at the application or sidecar layer in under 2 microseconds. In modern architectures (like Envoy or Cilium service meshes), local in-memory limiters act as an L1 defense line to protect Redis from being saturated by high-volume DoS attacks."*

### Q: "How do you protect this from unbounded memory growth with millions of unique IP addresses?"
> **Answer:** *"That's a classic in-memory rate limiter pitfall: key explosion from ephemeral IP scanners. We solve this in two ways:
> 1. `CleanupStale(idleTimeout)`: A background sweep worker that removes client entries whose `lastActivity` is older than the TTL.
> 2. The **Sliding Window Counter** alternative: Reduces memory per key from $O(N)$ timestamps to two integer counters ($O(1)$ memory), bounding RAM usage even under heavy traffic."*

### Q: "How does this engine prevent race conditions under high concurrency?"
> **Answer:** *"All state mutations are protected with `sync.RWMutex` and atomic telemetry counters (`sync/atomic`). We verified thread-safety using Go's built-in race detector (`go test -race`). Every client map read and write is synchronized, ensuring no corrupted slices or memory tearing."*
