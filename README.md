# ugreen-WakeOnLan

UGREEN NAS 网络唤醒应用 — 通过魔术包（Magic Packet）远程唤醒局域网设备。

## 功能

- 管理设备列表（名称 + MAC 地址）
- 一键发送 WOL 魔术包唤醒设备
- 前后端分离：Go HTTP 后端 + 原生 HTML/JS 前端
- 支持 x86_64 和 ARM64 架构

## 项目结构

```
wakeonlan/
├── project.yaml            # UGREEN 应用配置 (spec v2.1)
├── main.go                 # Go 后端入口
├── go.mod / go.sum
├── rootfs_amd64/           # x86_64 编译产物
├── rootfs_arm64/           # ARM64 编译产物
└── rootfs_common/
    ├── icon.png            # 应用图标
    └── www/                # 前端静态文件
        ├── index.html
        └── app.js
```

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.23+, 标准库 (net/http) |
| 前端 | Vanilla HTML/CSS/JS |
| 打包 | ugcli v1.1.0.12 |

## 开发

### 环境

Docker 容器 `ugreen-go-dev` (Debian 12 + Go 1.26.3):

```bash
cd ~/Codes/WakeOnLan
./dev.sh              # 进入容器
```

### 编译

```bash
# x86_64
CGO_ENABLED=0 go build -o rootfs_amd64/wakeonlan_serv .

# ARM64 交叉编译
GOARCH=arm64 CGO_ENABLED=0 go build -o rootfs_arm64/wakeonlan_serv .
```

### 校验 & 打包

```bash
ugcli check   # 校验 project.yaml
ugcli pack    # 生成 .upk 安装包
```

### 本地测试

```bash
go run . &
curl http://localhost:21010/api/devices
curl -X POST http://localhost:21010/api/wake -H 'Content-Type: application/json' -d '{"mac":"00:11:22:33:44:55"}'
```

## API

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/devices` | 获取设备列表 |
| POST | `/api/devices` | 添加设备 `{"name":"...","mac":"..."}` |
| POST | `/api/wake` | 发送魔术包 `{"mac":"..."}` |

## 前置条件

- UGREEN NAS 运行 UGOS Pro >= 1.13.0
- 开发者授权文件 `ugdev.sig`

## License

MIT
