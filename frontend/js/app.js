document.addEventListener('DOMContentLoaded', () => {
  const algoSelect = document.getElementById('algorithm-select');
  const clientKeyInput = document.getElementById('client-key');
  const clientTierSelect = document.getElementById('client-tier');
  const tierGroup = document.getElementById('tier-group');
  const costInput = document.getElementById('request-cost');
  const burstSlider = document.getElementById('burst-slider');
  const burstVal = document.getElementById('burst-val');
  const sendBtn = document.getElementById('send-btn');
  const burstSpamBtn = document.getElementById('burst-spam-btn');
  const resetBtn = document.getElementById('reset-btn');

  const statusDisplay = document.getElementById('status-display');
  const remainingDisplay = document.getElementById('remaining-display');
  const resetDisplay = document.getElementById('reset-display');
  const rejectedDisplay = document.getElementById('rejected-display');
  const windowTrack = document.getElementById('window-track');
  const headerCode = document.getElementById('header-code');
  const auditStream = document.getElementById('audit-stream');

  let totalRejected = 0;
  let activeSlots = [];

  // Handle algorithm toggle
  algoSelect.addEventListener('change', () => {
    tierGroup.style.display = algoSelect.value === 'multi_tier' ? 'block' : 'none';
  });

  // Slider change
  burstSlider.addEventListener('input', () => {
    burstVal.textContent = `${burstSlider.value} req${burstSlider.value > 1 ? 's' : ''}`;
  });

  // Send single or batch request
  async function fireRequest(count = 1) {
    const payload = {
      key: clientKeyInput.value.trim() || 'default-client',
      algorithm: algoSelect.value,
      cost: parseInt(costInput.value) || 1,
      tier: clientTierSelect.value
    };

    for (let i = 0; i < count; i++) {
      try {
        const response = await fetch('/api/allow', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify(payload)
        });

        const data = await response.json();
        updateUI(response.status, data);

        if (count > 1) {
          await new Promise(r => setTimeout(r, 60));
        }
      } catch (err) {
        console.error('Request failed:', err);
      }
    }
  }

  function updateUI(status, data) {
    const isAllowed = data.allowed;
    if (!isAllowed) totalRejected++;

    statusDisplay.textContent = isAllowed ? '200 OK' : '429 BLOCKED';
    statusDisplay.className = `metric-value ${isAllowed ? 'status-200' : 'status-429'}`;

    remainingDisplay.textContent = `${data.remaining} / ${data.limit}`;
    resetDisplay.textContent = `${(data.reset_after_sec || 0).toFixed(2)}s`;
    rejectedDisplay.textContent = totalRejected;

    // Update Header preview
    headerCode.textContent = `HTTP/1.1 ${isAllowed ? '200 OK' : '429 Too Many Requests'}
X-RateLimit-Limit: ${data.limit}
X-RateLimit-Remaining: ${data.remaining}
X-RateLimit-Reset: ${(data.reset_after_sec || 0).toFixed(2)}
X-RateLimit-Algorithm: ${data.algorithm}${!isAllowed ? `\nRetry-After: ${Math.ceil(data.reset_after_sec || 1)}` : ''}`;

    // Add visual window slot
    const slot = document.createElement('div');
    slot.className = `window-slot ${isAllowed ? '' : 'blocked'}`;
    windowTrack.appendChild(slot);
    activeSlots.push(slot);

    // Auto prune visual slots after 2 seconds
    setTimeout(() => {
      slot.remove();
      activeSlots = activeSlots.filter(s => s !== slot);
    }, 2000);

    // Append to audit log
    const now = new Date().toISOString().split('T')[1].slice(0, 12);
    const row = document.createElement('div');
    row.className = `audit-row ${isAllowed ? 'allowed' : 'rejected'}`;
    row.innerHTML = `
      <span>[${now}] <strong>${data.key}</strong> â€” ${data.algorithm}</span>
      <span>${isAllowed ? 'âœ… 200 OK (Rem: ' + data.remaining + ')' : 'â Œ 429 TOO MANY REQUESTS'}</span>
    `;

    auditStream.insertBefore(row, auditStream.firstChild);
    if (auditStream.children.length > 25) {
      auditStream.removeChild(auditStream.lastChild);
    }
  }

  sendBtn.addEventListener('click', () => fireRequest(parseInt(burstSlider.value)));
  burstSpamBtn.addEventListener('click', () => fireRequest(10));

  resetBtn.addEventListener('click', async () => {
    try {
      await fetch('/api/reset', { method: 'POST' });
      totalRejected = 0;
      rejectedDisplay.textContent = '0';
      remainingDisplay.textContent = '5 / 5';
      resetDisplay.textContent = '0.00s';
      statusDisplay.textContent = 'RESET';
      statusDisplay.className = 'metric-value';
      windowTrack.innerHTML = '';
      auditStream.innerHTML = '<div class="audit-row info-row">All internal limiter buffers cleared.</div>';
    } catch (e) {
      console.error(e);
    }
  });
});
