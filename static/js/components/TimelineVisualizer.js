// TimelineVisualizer.js — Visualizes active request timestamps moving through the sliding window

export class TimelineVisualizer {
  constructor(containerId, windowMs = 2000) {
    this.container = document.getElementById(containerId);
    this.windowMs = windowMs;
    this.dots = []; // { id, timestamp }
    this.render();
    this.startAnimationLoop();
  }

  render() {
    this.container.innerHTML = `
      <div class="card timeline-card">
        <div class="card-title">
          <span>Sliding Window Horizon (2.0s Active Window)</span>
          <span class="tag">Real-Time Timestamp Pruning</span>
        </div>
        <div style="font-size: 0.8rem; color: var(--text-dim); margin-top: -0.5rem;">
          Each dot is a timestamped request in RAM. Dots slide left as wall-clock time advances; crossing the yellow boundary prunes them and frees capacity.
        </div>
        <div class="timeline-viewport" id="timelineViewport">
          <div class="window-boundary">
            <span class="boundary-label">← 2.0s Boundary (Pruned & Purged)</span>
          </div>
          <div id="dotsContainer"></div>
        </div>
      </div>
    `;
    this.viewport = document.getElementById('timelineViewport');
    this.dotsContainer = document.getElementById('dotsContainer');
  }

  addTimestamp() {
    this.dots.push({
      id: Math.random().toString(36).substring(7),
      timestamp: Date.now()
    });
  }

  clear() {
    this.dots = [];
    if (this.dotsContainer) this.dotsContainer.innerHTML = '';
  }

  startAnimationLoop() {
    const loop = () => {
      const now = Date.now();
      const cutoff = now - this.windowMs;

      // Filter out dots older than window
      this.dots = this.dots.filter(d => d.timestamp > cutoff);

      if (this.dotsContainer) {
        this.dotsContainer.innerHTML = '';
        this.dots.forEach(dot => {
          const age = now - dot.timestamp;
          const pctFromRight = (age / this.windowMs) * 100;
          const leftPct = Math.max(0, 100 - pctFromRight);

          const el = document.createElement('div');
          el.className = 'timeline-dot';
          el.style.left = `${leftPct}%`;
          el.title = `Request at -${age}ms`;
          this.dotsContainer.appendChild(el);
        });
      }

      requestAnimationFrame(loop);
    };
    requestAnimationFrame(loop);
  }
}
