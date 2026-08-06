# ⏱️ Cowsay Timer for 408

为 408 考研学子打造的高颜值终端 TUI 计时与 Markdown 任务管理工具。

---

## 🎬 演示 (Demo)

| 横屏模式 (Landscape) | 竖屏/窄屏模式 (Portrait) |
| :---: | :---: |
| ![Landscape Demo](assets/demo.gif) | ![Portrait Demo](assets/demo2.gif) |
| [下载 `demo.cast`](assets/demo.cast) | [下载 `demo2.cast`](assets/demo2.cast) |

---

## ✨ 特性

- 🐮 **Cowsay 伴读 & 倒计时**：2026 考研实时倒计时与经典伴读奶牛。
- ⏱️ **ANSI 艺术字计时器**：高精度大字计时，支持计圈 (Lap) 与重置。
- 📋 **Markdown 任务管理**：自动同步 `MM-DD.md`，支持快捷勾选、添加、编辑、删除与文件切换。
- 🌌 **浩瀚夜空与盲文流星**：正弦波平滑呼吸繁星与 2x4 亚像素盲文光轨流星背景。
- 🎉 **100% 达成喷涌礼花**：全任务完成触发三向喷泉爆裂礼花与皇冠奶牛欢庆。

---

## 🛠️ 技术栈

- **Go** / **Bubble Tea** (TUI 框架) / **Lipgloss** (终端样式)

---

## 🚀 快速开始

```bash
# 编译与运行
go build -o stopwatch main.go
./stopwatch [tasks.md]
```

### 📦 单文件跨平台编译

```bash
# 打包 Linux 64 位单文件
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o stopwatch-linux main.go

# 打包 macOS (Apple Silicon M1/M2/M3) 单文件
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o stopwatch-macos-arm64 main.go

# 打包 Windows 64 位单文件 (.exe)
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o stopwatch.exe main.go
```

---

## ⌨️ 快捷键

| 快捷键 | 功能 |
| :--- | :--- |
| `Tab` | 切换窗口焦点 |
| `Space` / `s` | 启动 / 暂停计时 |
| `l` / `Enter` | 计圈 / 勾选任务 |
| `r` | 重置秒表 |
| `j` / `k` 或 `↓` / `↑` | 上下选择任务 |
| `Space` / `x` | 勾选 / 取消任务 |
| `a` / `e` / `d` | 添加 / 编辑 / 删除任务 |
| `n` | 切换计划文件 |
| `h` / `?` | 显示快捷键帮助 |
| `q` / `Ctrl+C` | 退出 |

---

## 📄 License

MIT License
