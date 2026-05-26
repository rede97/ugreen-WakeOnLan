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
      const iface = ifaces.find(i => i.name === dev.interface);
      const ips = iface ? iface.ips.join(', ') : dev.interface;
      return '<div class="row">' +
        '<div class="info">' +
          '<div class="name">' + esc(dev.name) + '</div>' +
          '<div class="detail">' + esc(dev.mac) + ' &middot; ' + esc(ips) + '</div>' +
        '</div>' +
        '<button class="btn btn-wake" title="Send Wake-on-LAN" ' +
          'onclick="wakeDevice(\'' + escAttr(dev.mac) + '\',\'' + escAttr(dev.interface) + '\')">&#9654;</button>' +
        '<button class="btn btn-del" title="Delete device" ' +
          'onclick="delDevice(\'' + escAttr(dev.name) + '\',\'' + escAttr(dev.mac) + '\',\'' + escAttr(dev.interface) + '\')">&#10005;</button>' +
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

(async function init() {
  await loadInterfaces();
  loadDevices();
})();
