<p align="center">
  <img src="assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">
  把自然语言变成可执行命令的 AI 终端助手
</p>

<p align="center">
  <strong>简体中文</strong> | <a href="docs/i18n/README.en.md">English</a> | <a href="docs/i18n/README.ja.md">日本語</a> | <a href="docs/i18n/README.ko.md">한국어</a> | <a href="docs/i18n/README.es.md">Español</a> | <a href="docs/i18n/README.fr.md">Français</a> | <a href="docs/i18n/README.de.md">Deutsch</a> | <a href="docs/i18n/README.ru.md">Русский</a> | <a href="docs/i18n/README.pt-br.md">Português (BR)</a>
</p>

<p align="center">
  <a href="https://github.com/BIBIYES/Hintly/stargazers">
    <img src="https://img.shields.io/github/stars/BIBIYES/Hintly?style=for-the-badge&logo=github&label=Stars&color=f5c542" alt="GitHub stars">
  </a>
  <a href="https://github.com/BIBIYES/Hintly/releases">
    <img src="https://img.shields.io/github/v/release/BIBIYES/Hintly?style=for-the-badge&logo=github" alt="Latest release">
  </a>
  <a href="https://github.com/BIBIYES/Hintly">
    <img src="https://img.shields.io/github/go-mod/go-version/BIBIYES/Hintly?style=for-the-badge&logo=go" alt="Go version">
  </a>
</p>

## 多语言目录

- 中文（主文档）：`README.md`
- English：`docs/i18n/README.en.md`
- 日本語：`docs/i18n/README.ja.md`
- 한국어：`docs/i18n/README.ko.md`
- Español：`docs/i18n/README.es.md`
- Français：`docs/i18n/README.fr.md`
- Deutsch：`docs/i18n/README.de.md`
- Русский：`docs/i18n/README.ru.md`
- Português (BR)：`docs/i18n/README.pt-br.md`

## 亮点

- 忘记命令不用再到处查，直接问 `hint`。
- 输入自然语言需求，返回可直接执行的终端命令。
- 自动注入环境上下文：`GOOS`、发行版、Shell、当前工作目录。
- TUI 交互体验：加载状态、命令高亮、重试、编辑、一键执行。
- 安全扫描：命中危险命令关键字时，执行前必须手动输入 `y`。

## 快速开始

1. 构建：

```bash
go mod tidy
go build ./cmd/hint
```

2. 初始化配置：

```bash
./hint -init
```

3. 直接提问：

```bash
./hint "查看 fail2ban sshd 封禁情况"
```

## Linux 一键安装/更新

安装最新版本（同一命令也用于更新）：

```bash
curl -fsSL https://raw.githubusercontent.com/BIBIYES/Hintly/main/scripts/install-linux.sh | bash
```

安装指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/BIBIYES/Hintly/main/scripts/install-linux.sh | VERSION=v1.0.3 bash
```

安装到用户目录（无 sudo 场景）：

```bash
curl -fsSL https://raw.githubusercontent.com/BIBIYES/Hintly/main/scripts/install-linux.sh | INSTALL_DIR="$HOME/.local/bin" bash
```

## 使用截图

![Hintly usage screenshot](assets/image.png)

## 快捷键

- `Enter`：执行当前命令。
- `r`：重试（追加“上一个建议不满意，请换一种实现方式”）。
- `e`：编辑命令后按 `Enter` 执行。
- `Esc` / `Ctrl+C`：取消并退出。

## 配置文件路径

- 所有系统统一：`~/.config/hint/config.yaml`
- Windows 对应路径：`%UserProfile%/.config/hint/config.yaml`

配置示例：

```yaml
base_url: https://api.openai.com/v1
api_key: sk-xxxx
model: gpt-4o
```

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
├── assets/
│   ├── logo.png
│   └── image.png
├── docs/
│   └── i18n/
│       ├── README.en.md
│       ├── README.ja.md
│       ├── README.ko.md
│       ├── README.es.md
│       ├── README.fr.md
│       ├── README.de.md
│       ├── README.ru.md
│       └── README.pt-br.md
├── go.mod
└── README.md
```
