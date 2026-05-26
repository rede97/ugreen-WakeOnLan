# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Wake-on-LAN app for UGREEN NAS (UGOS Pro). Native APP model — single Go binary with embedded web UI. Sends magic packets to wake devices on the LAN.

## Build & Dev

Development happens inside the Docker container defined at the workspace root:
```bash
./dev.sh              # enter container shell (pwd = /workspace = wakeonlan/)
./dev.sh exec <cmd>   # run command in container
```

Build inside container:
```bash
# From wakeonlan/ directory
CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_amd64/wakeonlan_serv .
GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_arm64/wakeonlan_serv .
```

`-ldflags="-s -w"` strips debug info (~31% smaller, ~6MB), `-trimpath` removes source paths.
Memory cap: server sets `GOMEMLIMIT=32MiB` + `GOGC=50` at startup. Override with env vars.

Packaging: `ugcli pack` (produces `.upk` in `build_dir/pkgs/upk/`).

Container: `ugreen-go-dev`, host network, mounts `wakeonlan/` to `/workspace`.

## Architecture

Single binary (`wakeonlan_serv`) with dual mode:
- **No args**: HTTP server on port 21010, serves `www/` (falls back to `rootfs_common/www/`) + REST API
- **With args**: CLI tool

Zero external Go dependencies — stdlib only.

### API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/devices` | List configured devices |
| POST | `/api/devices` | Add device `{name, mac, interface}` |
| DELETE | `/api/devices` | Delete device (match by all three fields) |
| GET | `/api/interfaces` | List network interfaces with IPs |
| POST | `/api/wake` | Send magic packet `{mac, interface}` |

Device config persisted to `devices.json` alongside the binary. Shared between CLI and HTTP modes.

### CLI Commands

```
wakeonlan_serv interfaces              List network interfaces
wakeonlan_serv list                    List configured devices
wakeonlan_serv add -name X -mac M -iface I
wakeonlan_serv delete -name X -mac M -iface I
wakeonlan_serv wake -name X            Wake by name (auto-fills MAC+iface)
wakeonlan_serv wake -mac M -iface I    Wake manually
```

### Magic Packet Sending

Looks up the named interface, gets its current IPv4 address, binds a UDP socket to that IP, and broadcasts to `255.255.255.255:9`.

### Directory Structure

```
wakeonlan/
├── project.yaml           # UGREEN app manifest (spec v2.1)
├── main.go                # Go backend entry point
├── go.mod / go.sum
├── devices.json           # Persistent config (created at runtime)
├── rootfs_amd64/          # Compiled binary for x86_64
├── rootfs_arm64/          # Compiled binary for ARM64
├── rootfs_common/
│   ├── icon.png           # ≥128x128 PNG
│   └── www/               # Frontend (served by Go server)
│       ├── index.html
│       └── app.js
└── build_dir/             # UPK packaging output
```

## UGREEN Constraints

- `app_id: com.mxq.wakeonlan` — immutable after first publish
- `open_type: inner` — desktop window mode, gateway injects auth headers
- `proxy_path: api` — gateway forwards `/api/*` to backend port 21010
- CGO_ENABLED=0 (static binary required)
- Binary name must match `start_cmd` in project.yaml (`./wakeonlan_serv`)
- `rootfs_amd64/` and `rootfs_arm64/` must each contain the compiled binary
- `rootfs_common/www/` holds all static frontend assets; `rootfs_common/icon.png` (128x128+) required
- UGREEN merges arch-specific rootfs with common rootfs at install time — at runtime, `www/` is alongside the binary, not under `rootfs_common/`
