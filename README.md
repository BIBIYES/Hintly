# hint

`hint` 是一个跨平台 AI 命令助手：输入自然语言需求，返回可直接执行的终端命令，并支持一键执行、重试与编辑。

## 功能

- `hint -init`：交互式初始化配置（`BaseURL`、`API Key`、`Model`）。
- 自动注入环境上下文：`GOOS`、发行版、Shell、当前工作目录。
- TUI 交互：加载 Spinner、命令高亮、快捷键操作。
- 安全扫描：命中危险命令关键字时，执行前必须手动输入 `y`。

## 目录结构

```text
hint/
├── cmd/
│   └── hint/
│       └── main.go
├── internal/
│   ├── ai/
│   ├── config/
│   ├── executor/
│   └── ui/
├── pkg/
│   └── sysinfo/
├── go.mod
└── README.md
```

## 安装依赖并构建

```bash
go mod tidy
go build ./cmd/hint
```

## 使用

1. 初始化：
./
```bash
./hint -init
```

2. 请求命令：

```bash
./hint "列出当前目录下最近 3 天修改过的 .go 文件"
```

## 快捷键

- `Enter`：执行当前命令。
- `r`：重试（追加“上一个建议不满意，请换一种实现方式”）。
- `e`：编辑命令后按 `Enter` 执行。
- `Esc` / `Ctrl+C`：取消并退出。

## 配置文件路径

默认配置路径：

- macOS/Linux：`~/.config/hint/config.yaml`（支持读取旧路径 `~/Library/Application Support/hint/config.yaml`）
- Windows：`%AppData%/hint/config.yaml`

配置示例：

```yaml
base_url: https://api.openai.com/v1
api_key: sk-xxxx
model: gpt-4o
```
