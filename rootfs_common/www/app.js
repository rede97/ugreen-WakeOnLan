// WakeOnLan Frontend
const API_BASE = '/api';

async function loadDevices() {
  try {
    const resp = await fetch(`${API_BASE}/devices`);
    const data = await resp.json();
    renderDevices(data.devices || []);
  } catch (e) {
    document.getElementById('device-list').innerHTML =
      '<div style="text-align:center;color:#f44;padding:20px;">Failed to load devices</div>';
  }
}

function renderDevices(devices) {
  const el = document.getElementById('device-list');
  if (devices.length === 0) {
    el.innerHTML = '<div style="text-align:center;color:#888;padding:20px;">No devices added yet</div>';
    return;
  }
  el.innerHTML = devices.map((d, i) => `
    <div class="device">
      <div>
        <div class="device-name">${esc(d.name)}</div>
        <div class="device-mac">${esc(d.mac)}</div>
      </div>
      <button onclick="wakeDevice('${esc(d.mac)}')">Wake</button>
    </div>
  `).join('');
}

async function wakeDevice(mac) {
  try {
    const resp = await fetch(`${API_BASE}/wake`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mac })
    });
    if (resp.ok) {
      alert('Magic packet sent!');
    } else {
      const err = await resp.json();
      alert('Error: ' + (err.error || 'Unknown'));
    }
  } catch (e) {
    alert('Network error: ' + e.message);
  }
}

async function addDevice() {
  const name = document.getElementById('name-input').value.trim();
  const mac = document.getElementById('mac-input').value.trim();
  if (!name || !mac) { alert('Please fill in both fields'); return; }
  try {
    const resp = await fetch(`${API_BASE}/devices`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, mac })
    });
    if (resp.ok) {
      document.getElementById('name-input').value = '';
      document.getElementById('mac-input').value = '';
      loadDevices();
    }
  } catch (e) {
    alert('Network error: ' + e.message);
  }
}

function esc(s) { return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/"/g,'&quot;'); }

loadDevices();
