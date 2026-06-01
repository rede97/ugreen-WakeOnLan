const API = '/api';
let ifaces = [];

function toast(msg, ok) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.className = 'toast ' + (ok ? 'toast-ok' : 'toast-err') + ' show';
  clearTimeout(el._tid);
  el._tid = setTimeout(() => { el.className = 'toast'; }, 2500);
}

async function loadInterfaces() {
  try {
    const r = await fetch(API + '/interfaces');
    const d = await r.json();
    ifaces = d.interfaces || [];
    const sel = document.getElementById('in-iface');
    sel.innerHTML = '<option value="">-- select interface --</option>' +
      ifaces.map(i => '<option value="' + esc(i.name) + '">' +
        esc(i.name) + ' (' + i.ips.join(', ') + ')</option>').join('');
  } catch (e) {
    toast('Failed to load network interfaces', false);
  }
}

async function loadDevices() {
  try {
    const r = await fetch(API + '/devices');
    const d = await r.json();
    const list = d.devices || [];
    const el = document.getElementById('device-list');
    if (list.length === 0) {
      el.innerHTML = '<div class="empty">No devices configured</div>';
      return;
    }
    el.innerHTML = list.map(dev => {
      return '<div class="device-card">' +
        '<div class="info">' +
          '<div class="name">' +
            '<span class="iface-badge">' + esc(dev.interface) + '</span>' +
            '<span class="dot">&middot;</span>' +
            esc(dev.name) +
          '</div>' +
          '<div class="mac">' + esc(dev.mac) + '</div>' +
        '</div>' +
        '<div class="btn-row">' +
          '<button class="btn btn-wake" title="Wake up" ' +
            'onclick="wakeDevice(\'' + escAttr(dev.mac) + '\',\'' + escAttr(dev.interface) + '\')">&#9654; Wake</button>' +
          '<button class="btn btn-del" title="Delete" ' +
            'onclick="delDevice(\'' + escAttr(dev.name) + '\',\'' + escAttr(dev.mac) + '\',\'' + escAttr(dev.interface) + '\')">&#10005;</button>' +
        '</div>' +
        '</div>';
    }).join('');
  } catch (e) {
    document.getElementById('device-list').innerHTML =
      '<div class="empty" style="color:#e74c3c">Failed to load devices</div>';
  }
}

async function addDevice() {
  const name = document.getElementById('in-name').value.trim();
  const mac = document.getElementById('in-mac').value.trim();
  const iface = document.getElementById('in-iface').value;
  if (!name || !mac || !iface) {
    toast('Please fill in all fields', false);
    return;
  }
  try {
    const r = await fetch(API + '/devices', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, mac, interface: iface })
    });
    if (r.ok) {
      document.getElementById('in-name').value = '';
      document.getElementById('in-mac').value = '';
      toast('Device added', true);
      loadDevices();
    } else {
      const e = await r.json();
      toast(e.error || 'Failed to add device', false);
    }
  } catch (e) {
    toast('Network error', false);
  }
}

async function delDevice(name, mac, iface) {
  if (!confirm('Delete device "' + name + '"?')) return;
  try {
    const r = await fetch(API + '/devices', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, mac, interface: iface })
    });
    if (r.ok) {
      toast('Device removed', true);
      loadDevices();
    } else {
      const e = await r.json();
      toast(e.error || 'Failed to delete', false);
    }
  } catch (e) {
    toast('Network error', false);
  }
}

async function wakeDevice(mac, iface) {
  try {
    const r = await fetch(API + '/wake', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mac, interface: iface })
    });
    if (r.ok) {
      toast('Magic packet sent successfully', true);
    } else {
      const e = await r.json();
      toast(e.error || 'Failed to send', false);
    }
  } catch (e) {
    toast('Network error', false);
  }
}

function esc(s) {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/"/g, '&quot;');
}

function escAttr(s) {
  return s.replace(/\\/g, '\\\\').replace(/'/g, "\\'");
}

async function arpScan() {
  const el = document.getElementById('arp-list');
  el.innerHTML = '<div class="empty">Scanning...</div>';
  try {
    const r = await fetch(API + '/arp');
    const d = await r.json();
    const entries = d.entries || [];
    if (entries.length === 0) {
      el.innerHTML = '<div class="empty">No ARP entries found</div>';
      return;
    }
    // Sort: configured devices first
    entries.sort((a, b) => (a.name ? 0 : 1) - (b.name ? 0 : 1));
    window._arpEntries = entries;

    el.innerHTML =
      '<table class="arp-table">' +
        '<thead><tr><th>IP</th><th>MAC</th><th>Device</th><th>Delay</th><th></th></tr></thead>' +
        '<tbody>' +
        entries.map((e, i) =>
          '<tr class="' + (e.name ? 'arp-device' : '') + '">' +
            '<td data-label="IP">' + esc(e.ip) + '</td>' +
            '<td data-label="MAC" class="arp-mac">' + esc(e.mac) + '</td>' +
            '<td data-label="Device" class="arp-name">' + (e.name || '-') + '</td>' +
            '<td data-label="Delay"><span id="ping-' + i + '">-</span></td>' +
            '<td data-label="" class="arp-action">' +
              '<button class="btn-ping" onclick="pingIP(' + i + ',\'' + escAttr(e.ip) + '\')">Ping</button></td>' +
          '</tr>'
        ).join('') +
        '</tbody>' +
      '</table>';
  } catch (e) {
    el.innerHTML = '<div class="empty" style="color:#e74c3c">ARP scan failed</div>';
  }
}

async function pingIP(idx, ip) {
  const span = document.getElementById('ping-' + idx);
  span.className = '';
  span.textContent = '...';
  try {
    const r = await fetch(API + '/ping', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ ip: ip })
    });
    const d = await r.json();
    if (d.alive) {
      span.className = 'arp-ping';
      span.textContent = d.latency;
    } else {
      span.className = 'arp-err';
      span.textContent = 'timeout';
      toast('Ping ' + ip + ': ' + (d.error || 'timeout'), false);
    }
  } catch (e) {
    span.className = 'arp-err';
    span.textContent = 'error';
    toast('Ping ' + ip + ': network error', false);
  }
}

(async function init() {
  await loadInterfaces();
  loadDevices();
})();
