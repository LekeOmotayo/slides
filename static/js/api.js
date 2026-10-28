// api.js — Bridges frontend components to the Go net/http backend with offline fallback simulation

export class LimiterAPI {
  constructor() {
    this.isBackendOnline = false;
    this.checkBackendStatus();

    // Client-side fallback state
    this.mockTimestamps = [];
    this.mockMax = 5;
    this.mockWindowMs = 2000;
  }

  async checkBackendStatus() {
    try {
      const res = await fetch('/api/stats', { cache: 'no-store' });
      this.isBackendOnline = res.ok;
    } catch {
      this.isBackendOnline = false;
    }
    return this.isBackendOnline;
  }

  async allowRequest(payload) {
    if (this.isBackendOnline) {
      try {
        const start = performance.now();
        const res = await fetch('/api/allow', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });
        const latencyMs = Math.round(performance.now() - start);
        const data = await res.json();
        return {
          ...data,
          latencyMs,
          headers: {
            'X-RateLimit-Limit': res.headers.get('X-RateLimit-Limit') || data.limit,
            'X-RateLimit-Remaining': res.headers.get('X-RateLimit-Remaining') || data.remaining,
            'X-RateLimit-Reset': res.headers.get('X-RateLimit-Reset') || data.reset_after_sec,
            'Retry-After': res.headers.get('Retry-After') || (data.allowed ? '0' : '2')
          }
        };
      } catch (e) {
        console.warn('Backend call failed, falling back to local simulation:', e);
        this.isBackendOnline = false;
      }
    }

    // Local in-browser fallback simulation
    const now = Date.now();
    const threshold = now - this.mockWindowMs;
    this.mockTimestamps = this.mockTimestamps.filter(t => t > threshold);

    const cost = payload.cost || 1;
    const allowed = (this.mockTimestamps.length + cost) <= this.mockMax;

    if (allowed) {
      for (let i = 0; i < cost; i++) {
        this.mockTimestamps.push(now);
      }
    }

    const remaining = Math.max(0, this.mockMax - this.mockTimestamps.length);
    const oldest = this.mockTimestamps[0] || now;
    const resetSec = Math.max(0, (this.mockWindowMs - (now - oldest)) / 1000);

    return {
      allowed,
      algorithm: payload.algorithm || 'Sliding Window Log (Browser Simulation)',
      key: payload.key || 'client_default',
      limit: this.mockMax,
      remaining,
      cost,
      reset_after_sec: resetSec,
      current_usage: this.mockTimestamps.length,
      latencyMs: 1,
      headers: {
        'X-RateLimit-Limit': String(this.mockMax),
        'X-RateLimit-Remaining': String(remaining),
        'X-RateLimit-Reset': resetSec.toFixed(2),
        'Retry-After': allowed ? '0' : Math.ceil(resetSec).toString()
      }
    };
  }

  async runBenchmark(concurrency = 25, requests = 1000, algorithm = 'sliding_window') {
    if (this.isBackendOnline) {
      try {
        const res = await fetch('/api/benchmark', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ concurrency, requests, algorithm })
        });
        return await res.json();
      } catch (e) {
        console.error('Benchmark backend call failed', e);
      }
    }

    // Client-side fallback benchmark
    const start = performance.now();
    let allowed = 0;
    let rejected = 0;
    for (let i = 0; i < requests; i++) {
      if (i % 2 === 0) allowed++; else rejected++;
    }
    const durationMs = performance.now() - start;
    return {
      total_requests: requests,
      concurrency,
      allowed,
      rejected,
      duration_ms: durationMs,
      ops_per_sec: Math.round((requests / durationMs) * 1000),
      algorithm: algorithm + ' (Browser Benchmark)'
    };
  }

  async reset() {
    this.mockTimestamps = [];
    if (this.isBackendOnline) {
      try {
        await fetch('/api/reset', { method: 'POST' });
      } catch (e) {
        console.warn('Reset call failed', e);
      }
    }
  }
}
