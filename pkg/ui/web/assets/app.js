'use strict';

// ─── State ────────────────────────────────────────────────────────────────────
const state = {
  services: [],
  summary:  null,
  filter:   'all',
  view:     'dashboard',
  countdown: 30,
  timer:    null,
};

// ─── API ──────────────────────────────────────────────────────────────────────
async function apiFetch(path) {
  const res = await fetch(path);
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json();
}

async function loadAll() {
  const [services, summary, ver] = await Promise.allSettled([
    apiFetch('/api/platform/services'),
    apiFetch('/api/platform/summary'),
    apiFetch('/api/version'),
  ]);

  if (services.status === 'fulfilled') {
    state.services = services.value || [];
    renderGrid();
    renderTable();
    renderGroupFilter();
  }

  if (summary.status === 'fulfilled') {
    state.summary = summary.value;
    renderSummary();
    renderOverallStatus();
  }

  if (ver.status === 'fulfilled') {
    const v = ver.value;
    document.getElementById('coreVersion').textContent = v.version || 'dev';
  }
}

async function triggerPoll() {
  const btn = document.getElementById('refreshBtn');
  btn.classList.add('spinning');
  try {
    await apiFetch('/api/platform/poll');
    await new Promise(r => setTimeout(r, 600)); // brief pause
    await loadAll();
    resetCountdown();
    toast('Refreshed');
  } catch(e) {
    toast('Refresh failed: ' + e.message);
  } finally {
    btn.classList.remove('spinning');
  }
}

// ─── Countdown ────────────────────────────────────────────────────────────────
function resetCountdown() {
  state.countdown = 30;
  clearInterval(state.timer);
  state.timer = setInterval(() => {
    state.countdown--;
    document.getElementById('countdown').textContent = state.countdown + 's';
    if (state.countdown <= 0) {
      triggerPoll();
    }
  }, 1000);
}

// ─── Rendering ────────────────────────────────────────────────────────────────
function statusClass(s) {
  if (!s) return 'unknown';
  if (s === 'ok') return 'ok';
  if (s === 'degraded') return 'degraded';
  if (s === 'down' || s === 'error') return 'down';
  return 'unknown';
}

function statusLabel(s) {
  if (!s || s === 'unknown') return 'Unknown';
  if (s === 'ok') return 'Healthy';
  if (s === 'degraded') return 'Degraded';
  if (s === 'down' || s === 'error') return 'Down';
  return s;
}

function renderSummary() {
  const s = state.summary;
  if (!s) return;
  document.getElementById('sumTotal').textContent   = s.services  ?? '—';
  document.getElementById('sumHealthy').textContent = s.healthy   ?? '—';
  document.getElementById('sumDegraded').textContent= s.degraded  ?? '—';
  document.getElementById('sumDown').textContent    = s.down      ?? '—';
}

function renderOverallStatus() {
  const s = state.summary;
  const el = document.getElementById('overallStatus');
  const sc = s ? statusClass(s.status) : 'unknown';
  el.innerHTML = `<span class="status-dot ${sc}"></span><span class="status-label">${statusLabel(s?.status)}</span>`;
}

function groups() {
  const all = new Set(state.services.map(s => s.group).filter(Boolean));
  return ['all', ...all];
}

function renderGroupFilter() {
  const el = document.getElementById('groupFilter');
  const gs = groups();
  if (gs.length <= 2) { el.innerHTML = ''; return; }
  el.innerHTML = gs.map(g => `
    <button class="filter-chip${state.filter === g ? ' active' : ''}" data-group="${g}">
      ${g === 'all' ? 'All' : capitalize(g)}
    </button>
  `).join('');
  el.querySelectorAll('.filter-chip').forEach(btn =>
    btn.addEventListener('click', () => {
      state.filter = btn.dataset.group;
      renderGroupFilter();
      renderGrid();
    })
  );
}

function visibleServices() {
  return state.filter === 'all'
    ? state.services
    : state.services.filter(s => s.group === state.filter);
}

function renderGrid() {
  const grid = document.getElementById('servicesGrid');
  const svcs = visibleServices();

  if (!svcs.length) {
    grid.innerHTML = '<p style="color:var(--text-muted);grid-column:1/-1;padding:40px;text-align:center;">No services match the current filter.</p>';
    return;
  }

  grid.innerHTML = svcs.map(s => {
    const sc = statusClass(s.status);
    const checks = s.checks && Object.keys(s.checks).length > 0 ? s.checks : null;

    const checksHtml = checks ? `
      <div class="checks-list">
        ${Object.entries(checks).map(([k, v]) => `
          <div class="check-row">
            <span class="check-name">${escHtml(k)}</span>
            <span class="check-val ${v === 'ok' ? 'ok' : 'err'}">${v === 'ok' ? '✓ ok' : escHtml(v.slice(0, 60))}</span>
          </div>
        `).join('')}
      </div>
    ` : '';

    const ago = s.last_check ? timeAgo(s.last_check) : '—';
    const lat = s.latency_ms ? `${s.latency_ms}ms` : '—';

    return `
      <div class="service-card ${sc}">
        <div class="card-header">
          <div>
            <div class="card-name">${escHtml(s.display_name || s.name)}</div>
            ${s.group ? `<div class="card-group">${escHtml(s.group)}</div>` : ''}
          </div>
          <div class="card-status ${sc}">
            <span class="status-dot ${sc}"></span>
            ${statusLabel(s.status)}
          </div>
        </div>
        ${s.description ? `<div class="card-desc">${escHtml(s.description)}</div>` : ''}
        <div class="card-meta">
          <span class="meta-item">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M12 6v6l4 2"/></svg>
            ${ago}
          </span>
          <span class="meta-item">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M13 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V9z"/><polyline points="13 2 13 9 20 9"/></svg>
            ${escHtml(s.version || '—')}
          </span>
          <span class="meta-item">
            <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>
            ${lat}
          </span>
        </div>
        ${checksHtml}
        ${s.error ? `<div style="margin-top:10px;font-size:11.5px;color:var(--down);font-family:monospace;">${escHtml(s.error.slice(0,120))}</div>` : ''}
      </div>
    `;
  }).join('');
}

function renderTable() {
  const tbody = document.getElementById('servicesTableBody');
  tbody.innerHTML = state.services.map(s => {
    const sc = statusClass(s.status);
    const lat = s.latency_ms || 0;
    const latClass = lat < 100 ? 'fast' : lat < 500 ? 'medium' : 'slow';
    const ago = s.last_check ? timeAgo(s.last_check) : '—';

    return `<tr>
      <td><strong>${escHtml(s.display_name || s.name)}</strong></td>
      <td><span style="color:var(--text-muted)">${escHtml(s.group || '—')}</span></td>
      <td><span class="table-status ${sc}"><span class="status-dot ${sc}"></span>${statusLabel(s.status)}</span></td>
      <td><span style="font-family:monospace;font-size:12px">${escHtml(s.version || '—')}</span></td>
      <td><span class="latency-badge ${latClass}">${lat ? lat+'ms' : '—'}</span></td>
      <td><span style="color:var(--text-muted);font-size:12px">${ago}</span></td>
      <td><span class="url-cell">${escHtml(s.url || '—')}</span></td>
    </tr>`;
  }).join('');
}

// ─── View switching ───────────────────────────────────────────────────────────
function showView(name) {
  state.view = name;
  document.querySelectorAll('.view').forEach(v => v.classList.remove('active'));
  document.getElementById('view-' + name).classList.add('active');
  document.querySelectorAll('.nav-item').forEach(a => {
    a.classList.toggle('active', a.dataset.view === name);
  });
  const titles = { dashboard: 'Dashboard', services: 'Services' };
  document.getElementById('pageTitle').textContent = titles[name] || name;
}

// ─── Toast ────────────────────────────────────────────────────────────────────
function toast(msg) {
  const el = document.createElement('div');
  el.className = 'toast';
  el.textContent = msg;
  document.getElementById('toasts').appendChild(el);
  setTimeout(() => el.remove(), 3000);
}

// ─── Utils ───────────────────────────────────────────────────────────────────
function escHtml(s) {
  return String(s)
    .replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;')
    .replace(/"/g,'&quot;');
}

function capitalize(s) {
  return s.charAt(0).toUpperCase() + s.slice(1);
}

function timeAgo(iso) {
  try {
    const d = new Date(iso);
    const secs = Math.floor((Date.now() - d) / 1000);
    if (secs < 5)  return 'just now';
    if (secs < 60) return secs + 's ago';
    if (secs < 3600) return Math.floor(secs/60) + 'm ago';
    return Math.floor(secs/3600) + 'h ago';
  } catch { return '—'; }
}

// ─── Boot ─────────────────────────────────────────────────────────────────────
document.addEventListener('DOMContentLoaded', () => {
  // Navigation
  document.querySelectorAll('.nav-item[data-view]').forEach(a =>
    a.addEventListener('click', e => { e.preventDefault(); showView(a.dataset.view); })
  );

  // Refresh button
  document.getElementById('refreshBtn').addEventListener('click', triggerPoll);

  // Theme toggle
  const themeBtn = document.getElementById('themeToggle');
  themeBtn.addEventListener('click', () => {
    const curr = document.documentElement.getAttribute('data-theme');
    const next = curr === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    try { localStorage.setItem('params-theme', next); } catch {}
  });
  try {
    const saved = localStorage.getItem('params-theme');
    if (saved) document.documentElement.setAttribute('data-theme', saved);
  } catch {}

  // Initial load
  loadAll().catch(err => {
    console.error('Initial load failed:', err);
    document.getElementById('servicesGrid').innerHTML =
      `<p style="color:var(--down);grid-column:1/-1;padding:40px;text-align:center;">
        Could not reach the dashboard API: ${escHtml(err.message)}
      </p>`;
  });

  resetCountdown();
});
