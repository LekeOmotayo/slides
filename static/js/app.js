// app.js — Main application orchestrator for Leke Sliding Window Limiter Workbench

import { LimiterAPI } from './api.js';
import { TrafficGenerator } from './components/TrafficGenerator.js';
import { QuotaGauge } from './components/QuotaGauge.js';
import { TimelineVisualizer } from './components/TimelineVisualizer.js';
import { HeaderInfo } from './components/HeaderInfo.js';
import { BenchmarkPanel } from './components/BenchmarkPanel.js';

class LimiterApp {
  constructor() {
    this.api = new LimiterAPI();
    this.totalAllowed = 0;
    this.totalBlocked = 0;

    this.init();
  }

  async init() {
    // Check backend connection
    const isOnline = await this.api.checkBackendStatus();
    const modeBadge = document.getElementById('engineModeBadge');
    if (modeBadge) {
      if (isOnline) {
        modeBadge.textContent = '🟢 Go Backend Active (Port 8003)';
        modeBadge.style.color = 'var(--emerald-success)';
      } else {
        modeBadge.textContent = '🟡 In-Browser Simulation Active';
        modeBadge.style.color = 'var(--amber-warning)';
      }
    }

    // Mount UI components
    this.gauge = new QuotaGauge('gaugeMount');
    this.timeline = new TimelineVisualizer('timelineMount', 2000);
    this.headers = new HeaderInfo('headersMount');
    
    this.generator = new TrafficGenerator('generatorMount', async (payload) => {
      await this.handleFireRequest(payload);
    });

    this.benchmark = new BenchmarkPanel('benchmarkMount', async (concurrency, requests) => {
      const algo = document.getElementById('algoSelect')?.value || 'sliding_window';
      const res = await this.api.runBenchmark(concurrency, requests, algo);
      this.logEvent(`[Benchmark Completed] ${concurrency} workers -> ${Math.round(res.ops_per_sec).toLocaleString()} ops/sec`, 'info');
      return res;
    });

    // Listen for reset events
    window.addEventListener('limiter:reset', async () => {
      await this.api.reset();
      this.timeline.clear();
      this.totalAllowed = 0;
      this.totalBlocked = 0;
      this.gauge.updateMetrics(0, 0);
      this.gauge.update({ limit: 5, remaining: 5, reset_after_sec: 0, current_usage: 0, key: 'prod_tenant_alpha' });
      this.logEvent('[Limiter Reset] All buckets and timestamps purged', 'info');
    });

    this.logEvent('[System Ready] Rate limiter initialized with sub-millisecond precision', 'info');
  }

  async handleFireRequest(payload) {
    const res = await this.api.allowRequest(payload);

    if (res.allowed) {
      this.totalAllowed++;
      this.timeline.addTimestamp();
      this.logEvent(`[${res.algorithm}] 🟢 200 OK — Key: ${res.key} (Used: ${res.current_usage}/${res.limit}, Reset: ${res.reset_after_sec.toFixed(2)}s)`, 'allowed');
    } else {
      this.totalBlocked++;
      this.logEvent(`[${res.algorithm}] 🔴 429 THROTTLED — Key: ${res.key} (Retry-After: ${res.headers['Retry-After']}s)`, 'blocked');
    }

    this.gauge.update(res);
    this.gauge.updateMetrics(this.totalAllowed, this.totalBlocked);
    this.headers.update(res);
  }

  logEvent(msg, type = 'info') {
    const logContainer = document.getElementById('eventLogStream');
    if (!logContainer) return;

    const timeStr = new Date().toLocaleTimeString();
    const entry = document.createElement('div');
    entry.className = 'log-entry';

    let colorClass = 'log-info';
    if (type === 'allowed') colorClass = 'log-allowed';
    if (type === 'blocked') colorClass = 'log-blocked';

    entry.innerHTML = `
      <span class="log-time">[${timeStr}]</span>
      <span class="${colorClass}">${msg}</span>
    `;

    logContainer.prepend(entry);

    // Keep log size bounded
    while (logContainer.children.length > 50) {
      logContainer.removeChild(logContainer.lastChild);
    }
  }
}

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
  window.app = new LimiterApp();
});
