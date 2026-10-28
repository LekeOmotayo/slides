// TrafficGenerator.js — Controls for triggering rate-limited API calls, bursts, and LLM token budgets

export class TrafficGenerator {
  constructor(containerId, onFire) {
    this.container = document.getElementById(containerId);
    this.onFire = onFire;
    this.render();
    this.bindEvents();
  }

  render() {
    this.container.innerHTML = `
      <div class="card">
        <div class="card-title">
          <span>Traffic Generator</span>
          <span class="tag">Simulate Client Traffic</span>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
          <div class="control-group">
            <label>Limiter Algorithm</label>
            <select id="algoSelect">
              <option value="sliding_window">Sliding Window Log (Exact O(N))</option>
              <option value="sliding_counter">Sliding Window Counter (Cloudflare O(1))</option>
              <option value="token_bucket">Token Bucket (LLM TPM / Bursts)</option>
            </select>
          </div>

          <div class="control-group">
            <label>Tenant / Client Key</label>
            <select id="keySelect">
              <option value="prod_tenant_alpha">prod_tenant_alpha</option>
              <option value="api_gateway_user_2">api_gateway_user_2</option>
              <option value="free_tier_guest">free_tier_guest</option>
            </select>
          </div>
        </div>

        <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
          <div class="control-group">
            <label>Cost / Tokens</label>
            <input type="number" id="costInput" min="1" max="10" value="1" />
          </div>

          <div class="control-group">
            <label>Tenant Tier</label>
            <select id="tierSelect">
              <option value="free">Free (5 tokens, 1/s)</option>
              <option value="pro">Pro (50 tokens, 10/s)</option>
              <option value="enterprise">Enterprise (500 tokens, 100/s)</option>
            </select>
          </div>
        </div>

        <div class="btn-group">
          <button id="btnFireOne" style="flex: 1;">⚡ Fire 1 Request</button>
          <button id="btnBurstFive" class="btn-secondary" style="flex: 1;">💥 Burst 5 Reqs</button>
          <button id="btnResetState" class="btn-danger">↺ Reset</button>
        </div>
      </div>
    `;
  }

  bindEvents() {
    const getPayload = (costOverride) => ({
      algorithm: document.getElementById('algoSelect').value,
      key: document.getElementById('keySelect').value,
      cost: costOverride || parseInt(document.getElementById('costInput').value, 10) || 1,
      tier: document.getElementById('tierSelect').value
    });

    document.getElementById('btnFireOne').addEventListener('click', () => {
      this.onFire(getPayload());
    });

    document.getElementById('btnBurstFive').addEventListener('click', async () => {
      for (let i = 0; i < 5; i++) {
        await this.onFire(getPayload());
        await new Promise(r => setTimeout(r, 40));
      }
    });

    document.getElementById('btnResetState').addEventListener('click', () => {
      window.dispatchEvent(new CustomEvent('limiter:reset'));
    });
  }
}
