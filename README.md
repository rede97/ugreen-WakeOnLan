# ugreen-WakeOnLan

UGREEN NAS 网络唤醒应用 -- 通过魔术包（Magic Packet）远程唤醒局域网设备。

v0.3.0

## 功能

- 管理设备列表（主机名 + MAC 地址 + 网口）
- 一键发送 WOL 魔术包唤醒设备
- ARP 扫描局域网设备（MAC-IP 映射，自动匹配已配置设备）
- ICMP Ping 延迟检测（DGRAM / Raw / 系统 ping 三级回退）
- 能力自检（权限不足时自动隐藏对应功能）
- 响应式 Web 界面（桌面 + 移动端）
- CLI 命令行完整支持
- 支持 x86_64 和 ARM64 架构

## 命令行

```bash
wakeonlan_serv interfaces                   列出网口及 IP
wakeonlan_serv list                         列出已配置设备
wakeonlan_serv add -name "PC" -mac "aa:bb:cc:dd:ee:ff" -iface eth0
wakeonlan_serv delete -name "PC" -mac "aa:bb:cc:dd:ee:ff" -iface eth0
wakeonlan_serv wake -name "PC"              按名字唤醒
wakeonlan_serv arp                          显示 ARP 表
wakeonlan_serv ping -ip 192.168.1.1         Ping 一个 IP
wakeonlan_serv scan                         ARP 扫描 + Ping 全部
wakeonlan_serv check                        检测 ARP/Ping 能力

# 不加参数启动 HTTP 服务（默认 :21010）
wakeonlan_serv
wakeonlan_serv -port 8080
```

## Web 界面

浏览器访问 `http://<nas>:21010`。

界面功能：
- 设备管理卡片：添加/删除设备，一键唤醒
- ARP 扫描卡片：Scan 按钮获取局域网 MAC-IP 列表，Ping 按钮测延迟
- 移动端自适应（480px 断点，卡片式布局）
- 能力检测：ARP/Ping 不可用时自动隐藏对应卡片

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 获取设备列表 |
| POST | `/api/devices` | 添加设备（返回 409 冲突） |
| DELETE | `/api/devices` | 删除设备 |
| GET | `/api/interfaces` | 列出网口及 IP |
| POST | `/api/wake` | 发送魔术包 |
| GET | `/api/arp` | ARP 扫描 `{entries, ping_ok, arp_ok}` |
| POST | `/api/ping` | Ping `{ip}` -> `{alive, latency}` |

设备配置持久化：`UGAPP_DATA_DIR` 环境变量 > 当前目录 `devices.json`。

## 开发

Docker 容器 `ugreen-go-dev` (Debian 12 + Go):

```bash
./dev.sh              # 进入容器 shell
./dev.sh exec <cmd>   # 在容器内执行命令
```

编译（需进入容器）：

```bash
CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_amd64/bin/wakeonlan_serv .
GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -ldflags="-s -w" -trimpath -o rootfs_arm64/bin/wakeonlan_serv .
```

图标转换（需 cairosvg）：

```bash
cairosvg icon.svg -o rootfs_common/icon.png -W 256 -H 256
```

一键打包：

```bash
./pack.sh N          # N 为 build 号
```

生成物在 `build_dir/pkgs/upk/`。

## 前置条件

- UGREEN NAS 运行 UGOS Pro >= 1.13.0
- 开发者授权文件 `ugdev.sig`
- ARP/Ping 功能在 UGOS Pro 沙箱环境中可能受限（详见 UGREEN_SUPPORT_EMAIL.txt）

## License

MIT
