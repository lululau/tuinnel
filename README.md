# ssh-tun-tui

基于终端的 SSH 隧道管理器，使用 Go、Bubble Tea 和 Bubbles 构建。提供多标签页 TUI 界面，支持通过 TOML 配置文件管理 SSH 隧道。

## 功能特性

- **三种隧道类型**：本地转发 (`-L`)、远程转发 (`-R`)、动态代理 (`-D`)
- **多标签页界面**：隧道列表、日志面板、全局设置、隧道编辑器
- **分组管理**：按分组批量启停隧道
- **实时日志**：查看每个隧道的 SSH 输出日志（环形缓冲区，最近 1000 行）
- **配置持久化**：TOML 格式配置文件，编辑后自动保存
- **SSH ControlMaster**：通过 Unix 控制套接字管理 SSH 连接生命周期

## 快速开始

### 安装

```bash
go build -o ssh-tun-tui .
```

### 配置

首次运行会自动创建默认配置文件：

```
~/.config/ssh-tun-tui/config.toml
```

或手动复制示例配置：

```bash
mkdir -p ~/.config/ssh-tun-tui
cp examples/config.toml ~/.config/ssh-tun-tui/config.toml
```

### 运行

```bash
./ssh-tun-tui
```

## 配置说明

```toml
[settings]
ssh_bin = "ssh"                  # SSH 二进制文件路径
control_dir = "/tmp/ssh-tun-tui" # 控制套接字目录
kill_on_exit = false              # 退出时是否关闭所有隧道

# 本地端口转发
[[tunnels]]
name = "dev-db"
type = "local"           # local | remote | dynamic
local_port = 3307
remote_host = "localhost"
remote_port = 3306
login = "deploy@db-server"
group = "dev"            # 可选，用于分组管理
```

### 隧道类型

| 类型 | 标志 | 转发规则 |
|------|------|----------|
| 本地转发 | `-L` | `local_port:remote_host:remote_port` |
| 远程转发 | `-R` | `remote_port:remote_host:local_port` |
| 动态代理 | `-D` | `local_port`（SOCKS 代理） |

## 界面布局

```
┌─ ssh-tun-tui ────────────────────────────────────┐
│ [隧道] [日志] [设置] [编辑器]          ?=帮助    │
├──────────────────────────────────────────────────┤
│                                                   │
│              (当前标签页内容)                       │
│                                                   │
├──────────────────────────────────────────────────┤
│ 1/4 隧道运行中 │ 分组: dev │ j/k: 上下移动       │
└───────────────────────────────────────────────────┘
```

### 标签页

- **隧道列表**：表格视图，显示隧道状态、名称、类型、端口等信息
- **日志面板**：左侧紧凑隧道列表 + 右侧选中隧道的 SSH 日志（可滚动）
- **全局设置**：SSH 路径、控制套接字目录、退出行为
- **隧道编辑器**：新增/编辑/删除隧道配置

## 快捷键

### 全局

| 按键 | 功能 |
|------|------|
| `1`-`4` / `Tab` / `Shift+Tab` | 切换标签页 |
| `q` | 退出（隧道运行时需确认） |
| `?` | 帮助对话框 |

### 隧道列表

| 按键 | 功能 |
|------|------|
| `Enter` / `r` | 启动隧道 |
| `k` | 停止隧道 |
| `R` | 重启隧道 |
| `g` | 刷新状态 |
| `e` | 在编辑器中编辑 |
| `/` | 搜索过滤 |
| `j` / `↑` | 上一条 |
| `J` / `↓` | 下一条 |

## 项目结构

```
ssh-tun-tui/
├── main.go                    # 入口，Bubble Tea 初始化
├── internal/
│   ├── app/
│   │   └── model.go           # 顶层 App Model，标签页切换
│   ├── tunnel/
│   │   ├── tunnel.go          # Tunnel 结构体定义
│   │   ├── manager.go         # SSH 隧道生命周期管理
│   │   └── config.go          # TOML 配置读写
│   ├── ui/
│   │   ├── styles.go          # 全局 Lipgloss 主题样式
│   │   └── tabs/
│   │       ├── tablist.go     # 标签栏组件
│   │       ├── tunnel_list.go # 隧道列表标签页
│   │       ├── logs.go        # 日志面板标签页
│   │       ├── settings.go    # 全局设置标签页
│   │       └── editor.go      # 隧道编辑器标签页
│   └── ssh/
│       └── client.go          # SSH 命令封装（ControlMaster 操作）
├── examples/
│   └── config.toml            # 示例配置文件
└── docs/
    └── plans/                 # 设计文档与实施计划
```

## 技术栈

- Go 1.23+
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) v2 — TUI 框架
- [Bubbles](https://github.com/charmbracelet/bubbles) v2 — 预构建 UI 组件
- [Lipgloss](https://github.com/charmbracelet/lipgloss) v2 — 样式引擎
- [BurntSushi/toml](https://github.com/BurntSushi/toml) — TOML 解析

## License

MIT
