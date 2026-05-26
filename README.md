# ugreen-WakeOnLan

UGREEN NAS 网络唤醒应用 — 通过魔术包（Magic Packet）远程唤醒局域网设备。

## 功能

- 管理设备列表（主机名 + MAC 地址 + 网口）
- 一键发送 WOL 魔术包唤醒设备
- Web 界面 + CLI 命令行两种模式
- 支持 x86_64 和 ARM64 架构

## 项目结构

```
├── project.yaml            # UGREEN 应用配置 (spec v2.1)
├── main.go                 # Go 后端入口
├── go.mod / go.sum
├── Dockerfile              # 开发容器
├── docker-compose.yml
├── dev.sh                  # 容器便捷脚本
├── rootfs_amd64/           # x86_64 编译产物
├── rootfs_arm64/           # ARM64 编译产物
└── rootfs_common/
    ├── icon.png            # 应用图标 (128x128+)
    └── www/                # 前端静态文件
        ├── index.html
        └── app.js
```

## 开发

Docker 容器 `ugreen-go-dev` (Debian 12 + Go 1.26.3):

```bash
./dev.sh              # 进入容器 shell
./dev.sh exec <cmd>   # 在容器内执行命令
```

编译:

```bash
CGO_ENABLED=0 go build -buildvcs=false -o rootfs_amd64/wakeonlan_serv .
GOARCH=arm64 CGO_ENABLED=0 go build -buildvcs=false -o rootfs_arm64/wakeonlan_serv .
```

校验与打包:

```bash
ugcli check   # 校验 project.yaml
ugcli pack    # 生成 .upk 安装包
```

## 使用方式

### Web 界面

直接运行启动 HTTP 服务，浏览器访问 `http://<nas>:21010`：

```bash
./wakeonlan_serv
```

### CLI 命令

```bash
wakeonlan_serv interfaces           # 列出网口及 IP
wakeonlan_serv list                 # 列出已配置设备
wakeonlan_serv add -name "PC" -mac "aa:bb:cc:dd:ee:ff" -iface eth0
wakeonlan_serv delete -name "PC" -mac "aa:bb:cc:dd:ee:ff" -iface eth0
wakeonlan_serv wake -name "PC"      # 按名字唤醒（自动查 MAC 和网口）
wakeonlan_serv wake -mac "aa:bb:cc:dd:ee:ff" -iface eth0
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 获取设备列表 |
| POST | `/api/devices` | 添加设备 `{name, mac, interface}` |
| DELETE | `/api/devices` | 删除设备（匹配所有字段） |
| GET | `/api/interfaces` | 列出网口及 IP |
| POST | `/api/wake` | 发送魔术包 `{mac, interface}` |

设备配置持久化到 `devices.json`。

## 前置条件

- UGREEN NAS 运行 UGOS Pro >= 1.13.0
- 开发者授权文件 `ugdev.sig`

## License

MIT
