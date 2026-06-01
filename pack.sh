#!/bin/bash
set -euo pipefail

BUILD="${1:-1}"
APP_ID="com.mxq.wakeonlan"
VERSION=$(grep '^version:' project.yaml | awk '{print $2}')

echo "=== WakeOnLan Pack ==="
echo "Version: ${VERSION}  Build: ${BUILD}"

# Clean
rm -rf build_dir

# Icon (SVG -> 256x256 PNG)
echo "[1/3] Generating icon..."
cairosvg icon.svg -o rootfs_common/icon.png -W 256 -H 256

# Build
echo "[2/3] Building amd64..."
CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_amd64/bin/wakeonlan_serv .
echo "      $(ls -lh rootfs_amd64/bin/wakeonlan_serv | awk '{print $5}')"

echo "[3/3] Building arm64..."
GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_arm64/bin/wakeonlan_serv .
echo "      $(ls -lh rootfs_arm64/bin/wakeonlan_serv | awk '{print $5}')"

# Pack
ugcli pack --build "$BUILD"

echo ""
echo "=== Done ==="
ls -lh build_dir/pkgs/upk/
