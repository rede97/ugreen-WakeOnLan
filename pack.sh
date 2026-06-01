#!/bin/bash
set -euo pipefail

BUILD="${1:-1}"
APP_ID="com.mxq.wakeonlan"
VERSION=$(grep '^version:' project.yaml | awk '{print $2}')

echo "=== WakeOnLan Pack ==="
echo "Version: ${VERSION}  Build: ${BUILD}"

# Clean
rm -rf build_dir

# Frontend
echo "[1/5] Building frontend..."
cd frontend
npm install --registry https://registry.npmjs.org --silent
npm run build
cd ..

# Icon (SVG -> 256x256 PNG)
echo "[2/5] Generating icon..."
cairosvg icon.svg -o rootfs_common/icon.png -W 256 -H 256

# Build
echo "[3/5] Building amd64..."
CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_amd64/bin/wakeonlan_serv .
echo "      $(ls -lh rootfs_amd64/bin/wakeonlan_serv | awk '{print $5}')"

echo "[4/5] Building arm64..."
GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_arm64/bin/wakeonlan_serv .
echo "      $(ls -lh rootfs_arm64/bin/wakeonlan_serv | awk '{print $5}')"

# Pack
echo "[5/5] Packing..."
ugcli pack --build "$BUILD"

echo ""
echo "=== Done ==="
ls -lh build_dir/pkgs/upk/
