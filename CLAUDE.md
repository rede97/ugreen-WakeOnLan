# WakeOnLan — UGREEN NAS APP

## Architecture
- **Backend**: Go 1.23+, single binary HTTP server (port 21010)
- **Frontend**: Vanilla HTML/CSS/JS (no framework), served from `rootfs_common/www/`
- **API**: `/api/devices` (GET list, POST add), `/api/wake` (POST send magic packet)
- **App model**: UGREEN Native APP (not Docker)
- **open_type**: `inner` (desktop window with JSSDK, gateway auth injection)

## Directory Structure
```
wakeonlan/
├── project.yaml           # UGREEN app manifest (spec v2.1)
├── main.go                # Go backend entry point
├── go.mod / go.sum
├── rootfs_amd64/          # Compiled binary for x86_64
├── rootfs_arm64/          # Compiled binary for ARM64
└── rootfs_common/
    ├── icon.png            # ≥128x128 PNG
    └── www/                # Served as static files by the Go server
        ├── index.html
        └── app.js
```

## Build
```bash
# Inside container: docker exec -w /workspace/wakeonlan ugreen-go-dev
CGO_ENABLED=0 go build -o rootfs_amd64/wakeonlan_serv .
GOARCH=arm64 CGO_ENABLED=0 go build -o rootfs_arm64/wakeonlan_serv .
```

## Validation
```bash
docker exec -w /workspace/wakeonlan ugreen-go-dev ugcli check
```

## project.yaml Key Rules
- `app_id`: immutable after first publish
- `version`: semver
- `permissions`: use strings like `NETWORK.ACCESS_INTERNET`, not maps
- `i18n`: locale codes must be like `en-US`, `zh-CN`; each needs `name`, `description`, `author`
- `depend_fw_version`: plain version like `1.13.0.0000`
- `start_cmd`: `./wakeonlan_serv`
- `proxy_path`: `api` → all `/api/*` requests forwarded to backend by UGREEN gateway
- `entry`: frontend entry relative to rootfs_common, e.g. `www/index.html`

## Auth (inner mode)
- System gateway forwards authenticated user info in HTTP headers to backend port 21010
- Frontend uses UGREEN JSSDK `getThirdToken()` for auth token

## Constraints
- Must compile with CGO_ENABLED=0 (static binary)
- Binary name must match `start_cmd` in project.yaml
- `rootfs_amd64/` and `rootfs_arm64/` MUST contain the compiled binary
- All static frontend assets go in `rootfs_common/www/`
- Icon must exist at `rootfs_common/icon.png` (128x128+)
- No external Go dependencies unless absolutely necessary (std lib preferred for WOL)

## Dev Container
```bash
cd ~/Codes/WakeOnLan
./dev.sh              # enter container shell
./dev.sh exec <cmd>   # run command in container
```
Container: `ugreen-go-dev` (Debian 12, Go 1.26.3, host network, GOPROXY=goproxy.cn)
