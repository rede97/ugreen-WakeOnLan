<script setup>
import { ref, onMounted } from 'vue'

const API = '/api'
const devices = ref([])
const ifaces = ref([])
const form = ref({ name: '', mac: '', iface: '' })
const toast = ref({ msg: '', ok: false, show: false })
let toastTimer = null

function showToast(msg, ok) {
  toast.value = { msg, ok, show: true }
  clearTimeout(toastTimer)
  toastTimer = setTimeout(() => { toast.value.show = false }, 2500)
}

async function loadInterfaces() {
  try {
    const r = await fetch(API + '/interfaces')
    const d = await r.json()
    ifaces.value = d.interfaces || []
  } catch {
    showToast('Failed to load network interfaces', false)
  }
}

async function loadDevices() {
  try {
    const r = await fetch(API + '/devices')
    const d = await r.json()
    devices.value = d.devices || []
  } catch {
    showToast('Failed to load devices', false)
  }
}

async function addDevice() {
  const { name, mac, iface: ifaceName } = form.value
  if (!name || !mac || !ifaceName) {
    showToast('Please fill in all fields', false)
    return
  }
  try {
    const r = await fetch(API + '/devices', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name, mac, interface: ifaceName }),
    })
    if (r.ok) {
      form.value = { name: '', mac: '', iface: '' }
      showToast('Device added', true)
      loadDevices()
    } else {
      const e = await r.json()
      showToast(e.error || 'Failed to add device', false)
    }
  } catch {
    showToast('Network error', false)
  }
}

async function delDevice(dev) {
  try {
    const r = await fetch(API + '/devices', {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: dev.name, mac: dev.mac, interface: dev.interface }),
    })
    if (r.ok) {
      showToast('Device removed', true)
      loadDevices()
    } else {
      const e = await r.json()
      showToast(e.error || 'Failed to delete', false)
    }
  } catch {
    showToast('Network error', false)
  }
}

async function wakeDevice(dev) {
  try {
    const r = await fetch(API + '/wake', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ mac: dev.mac, interface: dev.interface }),
    })
    if (r.ok) {
      showToast('Magic packet sent successfully', true)
    } else {
      const e = await r.json()
      showToast(e.error || 'Failed to send', false)
    }
  } catch {
    showToast('Network error', false)
  }
}

onMounted(async () => {
  await loadInterfaces()
  loadDevices()
})
</script>

<template>
  <div class="wrap">
    <h1>WakeOnLan</h1>

    <div class="card">
      <h3>Devices</h3>
      <div v-if="devices.length === 0" class="empty">No devices configured</div>
      <div v-for="d in devices" :key="d.mac" class="device-card">
        <div class="info">
          <div class="name">
            <span class="iface-badge">{{ d.interface }}</span>
            <span class="dot">&middot;</span>
            {{ d.name }}
          </div>
          <div class="mac">{{ d.mac }}</div>
        </div>
        <button class="btn btn-wake" title="Wake up" @click="wakeDevice(d)">&#9654;</button>
        <button class="btn btn-del" title="Delete" @click="delDevice(d)">&#10005;</button>
      </div>
    </div>

    <div class="card">
      <h3>Add Device</h3>
      <div class="form-row" style="margin-bottom:8px">
        <input v-model="form.name" placeholder="Hostname" style="flex:2" @keyup.enter="addDevice">
        <input v-model="form.mac" placeholder="MAC (AA:BB:CC:DD:EE:FF)" style="flex:2" @keyup.enter="addDevice">
      </div>
      <div class="form-row">
        <select v-model="form.iface">
          <option value="">-- select interface --</option>
          <option v-for="i in ifaces" :key="i.name" :value="i.name">
            {{ i.name }} ({{ i.ips.join(', ') }})
          </option>
        </select>
        <button class="btn btn-add" @click="addDevice">Add</button>
      </div>
    </div>

    <div class="card footer">
      <span class="ver">WakeOnLan v0.2.0</span>
      <span class="dot-sep">&middot;</span>
      <a href="https://github.com/rede97/ugreen-WakeOnLan" target="_blank" class="gh-link">GitHub</a>
    </div>
  </div>

  <div class="toast" :class="{ show: toast.show, 'toast-ok': toast.ok, 'toast-err': !toast.ok }">
    {{ toast.msg }}
  </div>
</template>

<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
  background: #1a1a2e;
  color: #eee;
  display: flex;
  justify-content: center;
  padding: 24px;
}
.wrap { max-width: 540px; width: 100%; }
h1 {
  text-align: center;
  margin-bottom: 24px;
  font-size: 1.4rem;
  color: #00d4aa;
  letter-spacing: 0.04em;
}
.card {
  background: #16213e;
  border-radius: 10px;
  padding: 16px;
  margin-bottom: 12px;
}
.card h3 {
  font-size: 0.95rem;
  color: #aaa;
  margin-bottom: 12px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
}
.empty {
  text-align: center;
  color: #666;
  padding: 16px 0;
  font-size: 0.9rem;
}
.footer { text-align: center; }
.ver { color: #888; font-size: 0.8rem; }
.dot-sep { color: #555; margin: 0 8px; }
.gh-link { color: #00d4aa; font-size: 0.8rem; text-decoration: none; }

.device-card {
  display: flex;
  align-items: center;
  gap: 0;
  background: #1a2240;
  border-radius: 8px;
  margin-bottom: 8px;
  padding: 12px;
}
.device-card .info { flex: 1; min-width: 0; }
.device-card .name { font-weight: 600; font-size: 0.95rem; }
.iface-badge {
  display: inline-block;
  background: #1f4068;
  color: #5db8fe;
  font-size: 0.72rem;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  margin-right: 4px;
  vertical-align: middle;
}
.dot { color: #555; margin: 0 4px; vertical-align: middle; font-size: 0.8rem; }
.device-card .mac {
  font-size: 0.78rem;
  color: #00d4aa;
  font-family: monospace;
  margin-top: 4px;
  padding-left: 2px;
}

.btn {
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 0.85rem;
  font-weight: 600;
  padding: 8px 14px;
  white-space: nowrap;
}
.btn-wake { background: #00d4aa; color: #1a1a2e; }
.btn-wake:hover { background: #00e6b8; }
.btn-del { background: none; color: #666; padding: 8px 10px; }
.btn-del:hover { color: #f55; }
.btn-add { background: #1f4068; color: #eee; flex: 0 0 auto; }
.btn-add:hover { background: #1b5090; }

.form-row { display: flex; gap: 8px; flex-wrap: wrap; }
.form-row input, .form-row select {
  padding: 8px 10px;
  border: 1px solid #1f3460;
  border-radius: 6px;
  background: #1a1a2e;
  color: #eee;
  font-size: 0.85rem;
  flex: 1;
  min-width: 100px;
}
.form-row select option { background: #16213e; color: #eee; }
input::placeholder { color: #555; }

.toast {
  position: fixed;
  top: 16px;
  left: 50%;
  transform: translateX(-50%);
  padding: 10px 24px;
  border-radius: 8px;
  font-size: 0.9rem;
  font-weight: 600;
  z-index: 999;
  opacity: 0;
  transition: opacity 0.3s;
  pointer-events: none;
}
.toast.show { opacity: 1; }
.toast-ok  { background: #00d4aa; color: #1a1a2e; }
.toast-err { background: #e74c3c; color: #fff; }

@media (max-width: 480px) {
  body { padding: 12px; }
  h1 { font-size: 1.2rem; margin-bottom: 16px; }
  .card { padding: 12px; margin-bottom: 8px; }
  .card h3 { font-size: 0.85rem; margin-bottom: 8px; }
  .device-card { padding: 10px; flex-wrap: wrap; gap: 6px; }
  .device-card .info { flex: 1 1 100%; }
  .device-card .name { font-size: 0.85rem; }
  .device-card .mac { font-size: 0.72rem; }
  .btn { font-size: 0.8rem; padding: 6px 10px; }
  .btn-wake { order: 1; }
  .btn-del { order: 2; padding: 6px 8px; }
  .form-row { gap: 6px; }
  .form-row input, .form-row select { padding: 7px 8px; font-size: 0.8rem; }
  .toast { width: 90%; text-align: center; font-size: 0.82rem; padding: 8px 16px; }
}
</style>
