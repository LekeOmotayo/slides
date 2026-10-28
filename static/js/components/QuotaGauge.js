// QuotaGauge.js — Visual meter showing real-time window capacity, tokens remaining, and reset countdown

export class QuotaGauge {
  constructor(containerId) {
    this.container = document.getElementById(containerId);
    this.limit = 5;
    this.remaining = 5;
    this.usage = 0;
    this.resetSec = 0;
    this.render();
    this.startTimer();
  }

  render() {
    this.container.innerHTML = `
      <div class="card">
        <div class="card-title">
          <span>Active Quota Status</span>
          <span class="tag" id="activeKeyTag">tenant: prod_tenant_alpha</span>
        </div>

        <div class="gauge-container">
          <div class="gauge-header">
            <span>Capacity Usage</span>
            <strong id="usageLabel" style="font-family: var(--font-mono);">${this.usage} / ${this.limit}</strong>
          </div>
          <div class="progress-track">
            <div id="progressBar" class="progress-bar"></div>
          </div>
          <div style="display: flex; justify-content: space-between; font-size: 0.8rem; color: var(--text-muted);">
            <span>Remaining: <strong id="remainingVal" style="color: var(--emerald-success);">${this.remaining}</strong></span>
            <span>Reset In: <strong id="resetVal" style="color: var(--amber-warning);">0.0s</strong></span>
          </div>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
          <div class="gauge-container" style="padding: 0.85rem;">
            <div style="font-size: 0.75rem; color: var(--text-dim);">LIFETIME ACCEPTED</div>
            <div id="metricAllowed" style="font-size: 1.4rem; font-weight: 700; color: var(--emerald-success); font-family: var(--font-mono);">0</div>
          </div>
          <div class="gauge-container" style="padding: 0.85rem;">
            <div style="font-size: 0.75rem; color: var(--text-dim);">LIFETIME THROTTLED (429)</div>
            <div id="metricBlocked" style="font-size: 1.4rem; font-weight: 700; color: var(--rose-danger); font-family: var(--font-mono);">0</div>
          </div>
        </div>
      </div>
    `;
  }

  update(res) {
    this.limit = res.limit || 5;
    this.remaining = res.remaining;
    this.usage = res.current_usage || (this.limit - this.remaining);
    this.resetSec = res.reset_after_sec || 0;

    const keyTag = document.getElementById('activeKeyTag');
    if (keyTag) keyTag.textContent = `tenant: ${res.key}`;

    const usageLabel = document.getElementById('usageLabel');
    if (usageLabel) usageLabel.textContent = `${this.usage} / ${this.limit}`;

    const remainingVal = document.getElementById('remainingVal');
    if (remainingVal) remainingVal.textContent = String(this.remaining);

    const resetVal = document.getElementById('resetVal');
    if (resetVal) resetVal.textContent = `${this.resetSec.toFixed(1)}s`;

    const bar = document.getElementById('progressBar');
    if (bar) {
      const pct = Math.min(100, Math.round((this.usage / this.limit) * 100));
      bar.style.width = `${pct}%`;
      if (pct >= 100) {
        bar.classList.add('danger');
      } else {
        bar.classList.remove('danger');
      }
    }
  }

  updateMetrics(allowed, blocked) {
    const ma = document.getElementById('metricAllowed');
    const mb = document.getElementById('metricBlocked');
    if (ma) ma.textContent = String(allowed);
    if (mb) mb.textContent = String(blocked);
  }

  startTimer() {
    setInterval(() => {
      if (this.resetSec > 0.1) {
        this.resetSec = Math.max(0, this.resetSec - 0.2);
        const resetVal = document.getElementById('resetVal');
        if (resetVal) resetVal.textContent = `${this.resetSec.toFixed(1)}s`;
      }
    }, 200);
  }
}
