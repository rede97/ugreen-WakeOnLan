# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Wake-on-LAN app for UGREEN NAS (UGOS Pro). Single Go binary with embedded web UI. Sends magic packets to wake devices on the LAN. Also scans ARP table and pings LAN hosts via native ICMP.

## Build & Dev

One-shot build + pack:
```bash
./pack.sh N
```
This runs: `cairosvg` icon generation → Go amd64 + arm64 → `ugcli pack`. UPK output in `build_dir/pkgs/upk/`.

```bash
# Quick local run
go run .

# Dev inside Docker container
./dev.sh              # enter container
./dev.sh exec <cmd>   # run command in container
```

Build flags: `CGO_ENABLED=0` (static binary), `-ldflags="-s -w"` (strip debug, ~6MB), `-trimpath`. Memory cap: `GOMEMLIMIT=32MiB` + `GOGC=50` set at HTTP server startup.

Icon: must be rendered with `cairosvg` (ImageMagick MSVG produces blank output):
```bash
cairosvg icon.svg -o rootfs_common/icon.png -W 256 -H 256
```

Zero Go dependencies — stdlib only, including ICMP ping (via `syscall.Socket` + raw `net.ListenPacket`).

## Architecture

Single binary `wakeonlan_serv` with two modes:
- **No args**: HTTP server on `:21010` (configurable via `-port`), serves static `www/` + REST API
- **CLI mode**: subcommands dispatched via `os.Args[1]`

### Source files

| File | Contents |
|------|----------|
| `main.go` | Entry point, CLI dispatch, HTTP server, device CRUD, magic packet, persistence |
| `arp.go` | `readArpTable()` reads `/proc/net/arp`, `handleArp` (API), `cmdArp` / `cmdScan` (CLI) |
| `ping.go` | 3-tier ICMP: DGRAM socket → raw socket → system `ping` command; `handlePing` (API), `pingICMP` |

Key package-level vars: `devices []Device`, `mu sync.Mutex`, `pingCapable bool`. `pingCapable` is set at server startup via self-test `ping 127.0.0.1`; exposed to frontend as `ping_ok` in `GET /api/arp` response.

### API Endpoints

| Method | Path | Purpose | Key fields |
|--------|------|---------|------------|
| GET | `/api/devices` | List devices | |
| POST | `/api/devices` | Add device | `{name, mac, interface}`, 409 on duplicate |
| DELETE | `/api/devices` | Delete device | Match by all three fields |
| GET | `/api/interfaces` | List NICs + IPs | |
| POST | `/api/wake` | Send magic packet | `{mac, interface}` |
| GET | `/api/arp` | Read ARP table | `{entries, ping_ok, arp_ok}` |
| POST | `/api/ping` | Ping an IP | `{ip}` → `{alive, latency}` |

### CLI Commands

```
wakeonlan_serv interfaces              List network interfaces
wakeonlan_serv list                    List configured devices
wakeonlan_serv add -name X -mac M -iface I
wakeonlan_serv delete -name X -mac M -iface I
wakeonlan_serv wake -name X            Wake by device name
wakeonlan_serv wake -mac M -iface I    Wake manually
wakeonlan_serv arp                     Show ARP table
wakeonlan_serv ping -ip 192.168.1.1    Ping an IP
wakeonlan_serv scan                    ARP scan + ping all entries
wakeonlan_serv check                   Report ARP/Ping capabilities
```

### Magic Packet

Constructed in `sendMagicPacket()`: 6×0xFF + 16×MAC, sent via UDP broadcast to `255.255.255.255:9` from the named interface's IPv4 address.

### Config Persistence

`devices.json` location:
- If `UGAPP_DATA_DIR` env is set → `$UGAPP_DATA_DIR/devices.json` (directory auto-created)
- Otherwise → `devices.json` alongside the binary

### Frontend

Static HTML/JS in `rootfs_common/www/`. Dark theme. Responsive breakpoint at 480px (mobile). Capability detection on page load: ARP card hidden by default, shown only after `GET /api/arp` returns `arp_ok: true`. Ping buttons rendered only when `ping_ok: true`. Script tag uses `app.js?v=X.Y.Z` for cache busting.

## UGREEN Constraints

- `app_id: com.mxq.wakeonlan` — immutable
- `open_type: inner` — desktop window mode, gateway injects auth headers
- `proxy_path: api` — `/api/*` forwarded to `:21010`
- `start_cmd: bin/wakeonlan_serv` (binary in `rootfs_{arch}/bin/`)
- `rootfs_common/www/` — static assets; `rootfs_common/icon.png` — 256×256 PNG
- CGO=0 required (static binary)
- Permissions: `NETWORK.ACCESS_INTERNET` + `SYSTEM.EXEC_SYSTEM_COMMAND` (for ping fallback)
- UGOS merges arch-specific rootfs with common rootfs at install. At runtime `www/` is alongside the binary.

## Known Platform Issues

On UGOS Pro, `/proc/net/arp` is not accessible (file not found) and all three ICMP ping methods fail (DGRAM socket likely blocked by seccomp, raw socket needs CAP_NET_RAW, system `/usr/bin/ping` not executable in sandbox). The app handles this gracefully via capability self-test on startup and `arp_ok`/`ping_ok` flags — frontend hides unavailable features.
