// BenchmarkPanel.js — Trigger real-time concurrent Go goroutine benchmarks

export class BenchmarkPanel {
  constructor(containerId, onRunBench) {
    this.container = document.getElementById(containerId);
    this.onRunBench = onRunBench;
    this.render();
    this.bindEvents();
  }

  render() {
    this.container.innerHTML = `
      <div class="card">
        <div class="card-title">
          <span>High-Concurrency Stress Test</span>
          <span class="tag">Go sync.Mutex Concurrency</span>
        </div>

        <div style="font-size: 0.85rem; color: var(--text-muted); line-height: 1.5;">
          Spawns concurrent workers simulating simultaneous incoming HTTP traffic against the in-memory rate limiter engine.
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
          <div class="control-group">
            <label>Concurrent Workers</label>
            <input type="number" id="benchConcurrency" value="25" min="5" max="100" />
          </div>
          <div class="control-group">
            <label>Total Requests</label>
            <input type="number" id="benchRequests" value="1000" min="100" max="10000" />
          </div>
        </div>

        <button id="btnRunBenchmark" class="btn-secondary">
          <span>🚀 Run Concurrency Benchmark</span>
        </button>

        <div id="benchResults" style="display: none; background: #060911; border: 1px solid var(--border-color); border-radius: 8px; padding: 0.85rem; font-family: var(--font-mono); font-size: 0.85rem;">
          <div style="display: flex; justify-content: space-between; margin-bottom: 0.4rem;">
            <span style="color: var(--text-dim);">Throughput:</span>
            <strong id="bThroughput" style="color: var(--cyan-primary);">0 ops/sec</strong>
          </div>
          <div style="display: flex; justify-content: space-between; margin-bottom: 0.4rem;">
            <span style="color: var(--text-dim);">Duration:</span>
            <span id="bDuration" style="color: var(--text-main);">0ms</span>
          </div>
          <div style="display: flex; justify-content: space-between;">
            <span style="color: var(--text-dim);">Accepted vs Throttled:</span>
            <span><strong id="bAllowed" style="color: var(--emerald-success);">0</strong> / <strong id="bRejected" style="color: var(--rose-danger);">0</strong></span>
          </div>
        </div>
      </div>
    `;
  }

  bindEvents() {
    const btn = document.getElementById('btnRunBenchmark');
    btn.addEventListener('click', async () => {
      const concurrency = parseInt(document.getElementById('benchConcurrency').value, 10) || 25;
      const requests = parseInt(document.getElementById('benchRequests').value, 10) || 1000;
      
      btn.disabled = true;
      btn.innerHTML = '<span>⏳ Running Go Goroutines...</span>';

      const res = await this.onRunBench(concurrency, requests);

      btn.disabled = false;
      btn.innerHTML = '<span>🚀 Run Concurrency Benchmark</span>';

      if (res) {
        document.getElementById('benchResults').style.display = 'block';
        document.getElementById('bThroughput').textContent = `${Math.round(res.ops_per_sec).toLocaleString()} ops/sec`;
        document.getElementById('bDuration').textContent = `${res.duration_ms.toFixed(2)} ms`;
        document.getElementById('bAllowed').textContent = String(res.allowed);
        document.getElementById('bRejected').textContent = String(res.rejected);
      }
    });
  }
}
