<p align="center">
  <img src="assets/logo.png" width="180" alt="Hintly logo">
</p>

<h1 align="center">Hintly</h1>

<p align="center">
  把自然语言变成可执行命令的 AI 终端助手
</p>

<p align="center">
  <strong>简体中文</strong> | <a href="docs/i18n/README.en.md">English</a>
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

## 文档

- 中文（主文档）：`README.md`
- English：`docs/i18n/README.en.md`

## 亮点

- 单次命令模式：`hint "你的需求"`。
- Agent 对话模式：仅输入 `hint` 进入多轮执行模式（思考 -> 执行 -> 读取结果 -> 再决策）。
- 自动注入环境上下文：`GOOS`、发行版、Shell、当前工作目录。
- 内置安全扫描：命中危险命令关键字会阻断执行。

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

3. 单次命令模式：

```bash
./hint "查看 fail2ban sshd 封禁情况"
```

4. Agent 对话模式：

```bash
./hint
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

## 单次命令模式快捷键

- `Enter`：执行当前命令。
- `r`：重试（会附带上一条不满意命令作为上下文）。
- `e`：编辑命令后按 `Enter` 执行。
- `Esc` / `Ctrl+C`：取消并退出。

## Agent 对话模式说明

- 输入目标后，Agent 会自动执行多步命令，直到完成或达到最大步数上限。
- 终端会显示每一步的思考、命令和命令输出。
- 输入 `exit` 或 `quit` 退出对话模式。

## 配置文件路径

- 所有系统统一：`~/.config/hint/config.yaml`
- Windows 对应路径：`%UserProfile%/.config/hint/config.yaml`

配置示例：

```yaml
base_url: https://api.openai.com/v1
api_key: sk-xxxx
model: gpt-4o
```
