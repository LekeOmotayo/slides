// HeaderInfo.js — Displays standard RFC 6585 / IETF RateLimit HTTP Response Headers

export class HeaderInfo {
  constructor(containerId) {
    this.container = document.getElementById(containerId);
    this.render();
  }

  render() {
    this.container.innerHTML = `
      <div class="card">
        <div class="card-title">
          <span>RFC HTTP RateLimit Headers</span>
          <span class="tag" id="statusBadge" style="color: var(--emerald-success); font-weight: 700;">200 OK</span>
        </div>

        <div class="headers-grid">
          <div class="header-box">
            <div class="header-name">X-RateLimit-Limit</div>
            <div class="header-val" id="hLimit">5</div>
          </div>
          <div class="header-box">
            <div class="header-name">X-RateLimit-Remaining</div>
            <div class="header-val" id="hRemaining">5</div>
          </div>
          <div class="header-box">
            <div class="header-name">X-RateLimit-Reset</div>
            <div class="header-val" id="hReset">0.00s</div>
          </div>
          <div class="header-box">
            <div class="header-name">Retry-After (429)</div>
            <div class="header-val" id="hRetry" style="color: var(--text-dim);">0s</div>
          </div>
        </div>

        <div style="display: flex; justify-content: space-between; font-size: 0.8rem; color: var(--text-dim); font-family: var(--font-mono);">
          <span>Algorithm: <span id="hAlgo" style="color: var(--cyan-primary);">Sliding Window Log</span></span>
          <span>Engine Latency: <span id="hLatency" style="color: var(--emerald-success);">&lt; 1ms</span></span>
        </div>
      </div>
    `;
  }

  update(res) {
    const statusBadge = document.getElementById('statusBadge');
    const hLimit = document.getElementById('hLimit');
    const hRemaining = document.getElementById('hRemaining');
    const hReset = document.getElementById('hReset');
    const hRetry = document.getElementById('hRetry');
    const hAlgo = document.getElementById('hAlgo');
    const hLatency = document.getElementById('hLatency');

    if (res.allowed) {
      statusBadge.textContent = '200 OK';
      statusBadge.style.color = 'var(--emerald-success)';
      hRemaining.className = 'header-val';
    } else {
      statusBadge.textContent = '429 TOO MANY REQUESTS';
      statusBadge.style.color = 'var(--rose-danger)';
      hRemaining.className = 'header-val throttled';
    }

    if (hLimit) hLimit.textContent = res.headers?.['X-RateLimit-Limit'] || res.limit;
    if (hRemaining) hRemaining.textContent = res.headers?.['X-RateLimit-Remaining'] || res.remaining;
    if (hReset) hReset.textContent = (res.headers?.['X-RateLimit-Reset'] || res.reset_after_sec.toFixed(2)) + 's';
    
    if (hRetry) {
      if (!res.allowed) {
        hRetry.textContent = (res.headers?.['Retry-After'] || Math.ceil(res.reset_after_sec)) + 's';
        hRetry.style.color = 'var(--rose-danger)';
      } else {
        hRetry.textContent = '0s';
        hRetry.style.color = 'var(--text-dim)';
      }
    }

    if (hAlgo) hAlgo.textContent = res.algorithm;
    if (hLatency) hLatency.textContent = `${res.latencyMs || 0.3}ms`;
  }
}
