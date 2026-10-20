# Leke Sliding Window Limiter & Token Budget Tracker

A thread-safe, zero-dependency Go rate limiter library implementing both **Sliding Window Log** and **Token Bucket** algorithms. Engineered to prevent API abuse and enforce token budget limits in high-concurrency microservices.

## Features
- **Pure Standard Library**: Zero external packages (`sync.Mutex`, `time`).
- **Thread-Safe**: Fully verified against Go race conditions (`go test -race`).
- **Sub-Millisecond Execution**: In-memory timestamp pruning and atomic token refill.

## Quick Start
```bash
# Run CLI demonstration
go run main.go

# Run unit & concurrency tests
go test -v -race ./...
```
